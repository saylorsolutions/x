package eventbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/env"
	"github.com/saylorsolutions/x/structures/queue"
	"github.com/saylorsolutions/x/structures/set"
	"github.com/saylorsolutions/x/syncx"
	"log"
	"sync"
	"time"
)

var (
	ErrNoHandler    = errors.New("no handler found")
	ErrInvalidEvent = errors.New("event ID 0 cannot be dispatched")
	ErrShuttingDown = errors.New("dispatch cannot be completed, event bus is shutting down")
)

// Event is a unique ID for an event in a domain.
// It's recommended to only use an [Event] ID for a specific purpose
// Do not use events [EventNone] or [EventAsyncError], as they are reserved for system use.
//
// If you want to listen for [EventAsyncError], then use [EventBus.RegisterErrorHandler].
type Event int

const (
	EventNone       Event = iota // EventNone is a reserved event used for detecting errors.
	EventAsyncError              // EventAsyncError is a reserved event used for transmitting processing errors.
)

var (
	instanceBus *EventBus
	initOnce    sync.Once
)

// InitInstance allows configuring the global [EventBus] instance.
// This function can only take effect once, so the return value should be checked to ensure that the intended configuration was applied.
func InitInstance(confFunc ...ConfigOption) bool {
	var didConfigure bool
	initOnce.Do(func() {
		instanceBus = NewEventBus(confFunc...)
		didConfigure = true
	})
	return didConfigure
}

// Instance is useful in cases where a single, global [EventBus] is desired.
// This can be helpful for accessing and synchronizing events while maintaining loose coupling.
// The global visibility means it's likely not a good fit for concurrent configuration, since it introduces the potential for configuration race conditions.
//
// This function doesn't allow configuring the global instance beyond the defaults. Use [InitInstance] to do that.
func Instance() *EventBus {
	initOnce.Do(func() {
		instanceBus = NewEventBus()
	})
	return instanceBus
}

type busConf struct {
	bufferSize   int
	numWorkers   int
	debugLogging bool
}

type ConfigOption func(conf *busConf) error

// OptBufferSize configures the [EventBus] to use the given size as the size of the dispatch buffer.
func OptBufferSize(size int) ConfigOption {
	return func(conf *busConf) error {
		if size < 1 {
			return fmt.Errorf("size '%d' is invalid, must be >= 1", size)
		}
		conf.bufferSize = size
		return nil
	}
}

// OptNumWorkers configures the [EventBus] to use the given num as the number of handler goroutines.
func OptNumWorkers(num int) ConfigOption {
	return func(conf *busConf) error {
		if num < 1 {
			return fmt.Errorf("num '%d' is invalid, must be >= 1", num)
		}
		conf.numWorkers = num
		return nil
	}
}

// OptEnableDebugLogging enables debug logging for this [EventBus].
//
// This can also be controlled by setting the environment variable EVENTBUS_DEBUG to a boolean value.
func OptEnableDebugLogging() ConfigOption {
	return func(conf *busConf) error {
		conf.debugLogging = true
		return nil
	}
}

// NewEventBus will create a new [EventBus] with default settings.
// ConfigFuncs may be used to specify different configuration parameters for the [EventBus].
// If none are specified, then both the dispatch buffer size and the number of handler goroutines will be set to [DefaultBufferSize].
func NewEventBus(opts ...ConfigOption) *EventBus {
	conf := busConf{
		bufferSize:   1,
		numWorkers:   1,
		debugLogging: env.Bool("EVENTBUS_DEBUG", false),
	}
	for _, fn := range opts {
		if err := fn(&conf); err != nil {
			panic(err)
		}
	}
	return &EventBus{
		handlers:      map[HandlerID]Handler{},
		handledEvents: map[Event]set.Set[HandlerID]{},
		conf:          conf,
	}
}

type Param any

func Paramf(format string, args ...any) Param {
	return Param(fmt.Sprintf(format, args...))
}

type HandlerID string

// Handler describes a component of the program that handles events received from the [EventBus].
type Handler interface {
	// HandleEvent will handle the given event and do some kind of processing.
	// Returned errors will be reported with a dispatched [EventAsyncError].
	HandleEvent(evt Event, params ...Param) error
	// Stop will alert the [Handler] that it should clean up resources and reject further events.
	// This can be ignored if not needed.
	Stop()
}

// HandlerFunc is a function that implements the Handler interface.
// This is intended for simple event handling cases where [Handler.Stop] has no real semantics for the handling component.
type HandlerFunc func(evt Event, params ...Param) error

func (f HandlerFunc) HandleEvent(evt Event, params ...Param) error {
	return f(evt, params...)
}

func (f HandlerFunc) Stop() {}

type busDispatch struct {
	event  Event
	params []Param
	future syncx.Future[error]
}

type EventBus struct {
	dispatchLoop    sync.Once
	stopDispatch    sync.Once
	doneDispatching sync.WaitGroup

	mux           sync.RWMutex
	events        *queue.ChannelQueue[*busDispatch]
	handlers      map[HandlerID]Handler
	handledEvents map[Event]set.Set[HandlerID]
	conf          busConf
}

func (b *EventBus) debug(args ...any) {
	if !b.conf.debugLogging {
		return
	}
	args = append([]any{"[EVENTBUS_DEBUG]"}, args...)
	log.Println(args...)
}

// Dispatch will submit an event to the [EventBus] for propagation.
// If an error occurs, then an [EventAsyncError] is propagated to an appropriate handler, if registered.
// If the EventBus is stopping, then this call will immediately return without dispatching.
//
// This can safely be called from within a [Handler].
func (b *EventBus) Dispatch(evt Event, params ...Param) {
	if evt == EventNone {
		b.DispatchError(ErrInvalidEvent)
		b.debug("no event specified for dispatch")
		return
	}
	dispatch := &busDispatch{
		event:  evt,
		params: params,
		future: syncx.SymbolicFuture[error](),
	}
	b.events.Push(dispatch)
	b.debug("event published to queue")
}

// DispatchResult will submit an event to the [EventBus] for propagation.
// If the [EventBus] is shutting down, then
// If an error is returned, then an [EventAsyncError] is still propagated to an appropriate handler, if registered.
//
// NOTE: This should not be called from within a [Handler], because it implicitly blocks a goroutine used for handling dispatches.
func (b *EventBus) DispatchResult(evt Event, params ...Param) syncx.Future[error] {
	if evt == EventNone {
		b.DispatchError(ErrInvalidEvent)
		b.debug("no event specified for dispatch")
		return syncx.StaticFuture(ErrInvalidEvent)
	}
	dispatch := &busDispatch{
		event:  evt,
		params: params,
		future: syncx.NewFuture[error](),
	}
	if !b.events.Push(dispatch) {
		dispatch.future.Resolve(ErrShuttingDown)
	}
	b.debug("event published to the queue, returning future")
	return dispatch.future
}

func (b *EventBus) DispatchErrorf(format string, args ...any) {
	b.DispatchError(fmt.Errorf(format, args...))
}

func (b *EventBus) DispatchError(err error) {
	b.Dispatch(EventAsyncError, err)
	b.debug("error published to queue")
}

func (b *EventBus) Register(id HandlerID, handledEvent Event, handler Handler) {
	syncx.LockFunc(&b.mux, func() {
		b.handlers[id] = handler
		b.handledEvents[handledEvent] = b.handledEvents[handledEvent].Add(id)
		b.debug("handler", id, "registered in the event bus to handle event", handledEvent)
	})
}

func (b *EventBus) RegisterFunc(id HandlerID, handledEvent Event, handler HandlerFunc) {
	b.Register(id, handledEvent, handler)
}

func (b *EventBus) RegisterErrorHandler(id HandlerID, handler func(error)) {
	b.Register(id, EventAsyncError, HandlerFunc(func(evt Event, params ...Param) error {
		var (
			err error
		)
		spec := ParamSpec(1,
			AssertAndStore(&err),
		)
		if errs := spec(params); len(errs) > 0 {
			b.debug("received invalid parameters in error handler dispatcher:", params)
			return fmt.Errorf("expected a single error parameter: %w", err)
		}
		handler(err)
		return nil
	}))
	b.debug("error handler registered")
}

func (b *EventBus) UnRegister(id HandlerID) {
	syncx.LockFunc(&b.mux, func() {
		handler, ok := b.handlers[id]
		if !ok {
			return
		}
		handler.Stop()
		delete(b.handlers, id)
		for _, handlerSet := range b.handledEvents {
			handlerSet.Remove(id)
		}
	})
	b.debug("unregistered handler", id)
}

func (b *EventBus) AddHandledEvent(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		_, ok := b.handlers[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		b.handledEvents[evt] = b.handledEvents[evt].Add(id)
		b.debug("handler", id, "updated to handle event", evt)
		return nil
	})
}

func (b *EventBus) SetHandledExclusive(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		_, ok := b.handlers[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		b.handledEvents[evt] = set.New(id)
		b.debug("handler", id, "updated to handle event", evt, "exclusively")
		return nil
	})
}

func (b *EventBus) RemoveHandledEvent(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		_, ok := b.handlers[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		b.handledEvents[evt] = b.handledEvents[evt].Remove(id)
		b.debug("handler", id, "updated to not handle event", evt)
		return nil
	})
}

// Start will start processing of dispatched events if it's not started already.
// Once the [EventBus] has stopped processing events, it cannot be restarted.
// This is safe to call multiple times from multiple goroutines. Only the first call to start will begin processing.
func (b *EventBus) Start(ctx context.Context) *EventBus {
	b.debug("EventBus.Start called")
	b.dispatchLoop.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		events, err := queue.NewChannelQueue[*busDispatch](ctx,
			queue.OptChannelSize(b.conf.numWorkers), queue.OptInitialBuffer(b.conf.bufferSize),
		)
		if err != nil {
			b.debug("error setting up channel queue:", err)
			// Shouldn't happen
			panic(err)
		}
		b.events = events
		b.doneDispatching.Add(b.conf.numWorkers)
		// Must cache events channel so a goroutine doesn't block after a call to Stop.
		for i := 0; i < b.conf.numWorkers; i++ {
			b.debug("starting event dispatch worker")
			go b.start(ctx, i, events)
		}
	})
	return b
}

func (b *EventBus) start(ctx context.Context, workerNum int, events *queue.ChannelQueue[*busDispatch]) {
	var debugLabel = fmt.Sprintf("[worker %d]", workerNum)
	defer b.doneDispatching.Done()
	defer func() {
		for _, handler := range b.handlers {
			handler.Stop()
		}
	}()
	var (
		errs  []error
		ctxCh = ctx.Done()
	)
	for {
		if len(errs) > 0 {
			b.debug(debugLabel, "dispatching errors from handlers")
			syncx.RLockFunc(&b.mux, func() {
				errHandlerIDs := b.handledEvents[EventAsyncError]
				if len(errHandlerIDs) == 0 {
					b.debug(debugLabel, "no registered error handlers")
					return
				}
				for _, err := range errs {
					for id := range errHandlerIDs {
						handler := b.handlers[id]
						if handler == nil {
							continue
						}
						if herr := handler.HandleEvent(EventAsyncError, err); herr != nil {
							b.debug(debugLabel, "error from error handler,", herr, ", while handling error:", err)
						}
					}
				}
			})
			errs = nil
		}
		select {
		case <-ctxCh:
			b.debug(debugLabel, "context cancelled, stopping dispatching worker")
			b.Stop()
			ctxCh = nil
		case dispatch, more := <-events.C:
			if !more {
				b.debug(debugLabel, "dispatch queue closed, stopping dispatching worker")
				return
			}
			b.debug(debugLabel, "received event for dispatching with event ", dispatch.event, "and params:", dispatch.params)
			syncx.RLockFunc(&b.mux, func() {
				defer func() {
					// If a result has already been returned or a result is not requested, then this does nothing
					dispatch.future.Resolve(nil)
				}()

				// Locate relevant handlers
				handlers := b.handledEvents[dispatch.event]
				noHandlersMessage := fmt.Errorf("%w for event %d", ErrNoHandler, dispatch.event)

				if len(handlers) == 0 {
					b.debug(debugLabel, "no handlers found for event")
					// Check if this is already an EventAsyncError
					if dispatch.event != EventAsyncError {
						dispatch.future.Resolve(noHandlersMessage)
						errs = append(errs, noHandlersMessage)
					}
					return
				}

				// Dispatch to all relevant handlers
				for id := range handlers {
					b.debug(debugLabel, "dispatching event to handler:", id)
					handler := b.handlers[id]
					if handler == nil {
						b.debug(debugLabel, "Handler no longer found! This is likely a bug in EventBus. Handler ID:", id)
						continue
					}
					err := handler.HandleEvent(dispatch.event, dispatch.params...)
					if err != nil {
						b.debug("handler", id, "returned error:", err)
						dispatch.future.Resolve(err)
						errs = append(errs, fmt.Errorf("handler '%s' failed to handle event %d: %v", id, dispatch.event, err))
					}
				}
			})
		}
	}
}

// Stop will stop the [EventBus] and immediately return without waiting for processing to complete in the background.
// To ensure that processing stops, call [EventBus.AwaitStop].
// This is safe to call multiple times from multiple goroutines if needed.
func (b *EventBus) Stop() {
	b.debug("EventBus.Stop called")
	b.stopDispatch.Do(func() {
		b.events.Stop()
	})
}

// AwaitStop will halt event processing for the [EventBus] if it's running, and wait for processing to stop.
// The given timeout value will be used to set a deadline for stopping.
// Calling this when the [EventBus] is already stopped will return immediately.
func (b *EventBus) AwaitStop(timeout time.Duration) {
	b.debug("EventBus.AwaitStop called")
	b.Stop()
	b.Await(timeout)
}

// Await will return when the [EventBus] is fully shut down.
// This can be useful when all logic in an application is made up of handlers and goroutines, and there is no blocking operation to prevent immediately exiting.
// If no timeout is specified, then this call will block until the start context is cancelled and all goroutines have exited.
func (b *EventBus) Await(timeout ...time.Duration) {
	b.debug("EventBus.Await called")
	b.events.Await()
	if len(timeout) > 0 && timeout[0] > 0 {
		_timeout := timeout[0]
		b.debug("awaiting shutdown with timeout:", _timeout.String())
		wait, cancel := context.WithTimeout(context.Background(), _timeout)
		go func() {
			defer cancel()
			b.doneDispatching.Wait()
		}()
		<-wait.Done()
	} else {
		b.debug("awaiting shutdown with no timeout")
		b.doneDispatching.Wait()
	}
	b.debug("done shutting down")
}
