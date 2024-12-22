package eventbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/structures/set"
	"github.com/saylorsolutions/x/syncx"
	"sync"
	"time"
)

var (
	ErrNoHandler    = errors.New("no handler found")
	ErrInvalidEvent = errors.New("event ID 0 cannot be dispatched")
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

const (
	// DefaultBufferSize dictates the default size of the dispatch channel, and the default number of handler workers.
	DefaultBufferSize = 1
)

var (
	instanceBus *EventBus
	initOnce    sync.Once
)

// InitInstance allows configuring the global [EventBus] instance.
// This function can only take effect once, so the return value should be checked to ensure that the intended configuration was applied.
func InitInstance(confFunc ...ConfigFunc) bool {
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
	bufferSize int
	numWorkers int
}

type ConfigFunc func(conf *busConf) error

// BufferSize configures the [EventBus] to use the given size as the size of the dispatch buffer.
func BufferSize(size int) ConfigFunc {
	return func(conf *busConf) error {
		if size < 1 {
			return fmt.Errorf("size '%d' is invalid, must be >= 1", size)
		}
		conf.bufferSize = size
		return nil
	}
}

// NumWorkers configures the [EventBus] to use the given num as the number of handler goroutines.
func NumWorkers(num int) ConfigFunc {
	return func(conf *busConf) error {
		if num < 1 {
			return fmt.Errorf("num '%d' is invalid, must be >= 1", num)
		}
		conf.numWorkers = num
		return nil
	}
}

// NewEventBus will create a new [EventBus] with default settings.
// ConfigFuncs may be used to specify different configuration parameters for the [EventBus].
// If none are specified, then both the dispatch buffer size and the number of handler goroutines will be set to [DefaultBufferSize].
func NewEventBus(configFunc ...ConfigFunc) *EventBus {
	conf := busConf{
		bufferSize: DefaultBufferSize,
		numWorkers: DefaultBufferSize,
	}
	for _, fn := range configFunc {
		if err := fn(&conf); err != nil {
			panic(err)
		}
	}
	return &EventBus{
		handlers:      map[HandlerID]Handler{},
		handledEvents: map[Event]set.Set[HandlerID]{},
		events:        make(chan *busDispatch, conf.bufferSize),
		numWorkers:    conf.numWorkers,
	}
}

type Param any

func Paramf(format string, args ...any) Param {
	return Param(fmt.Sprintf(format, args...))
}

type HandlerID string

// Handler describes a component of the program that handles events received from the [EventBus].
// Handlers should return quickly to prevent blocking other events, and may spin up additional goroutines to support this.
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
	numWorkers      int

	mux           sync.RWMutex
	events        chan *busDispatch
	handlers      map[HandlerID]Handler
	handledEvents map[Event]set.Set[HandlerID]
}

// Dispatch will submit an event to the [EventBus] for propagation.
// If an error occurs, then an [EventAsyncError] is propagated to an appropriate handler, if registered.
// If the EventBus is stopping, then this call will block indefinitely.
func (b *EventBus) Dispatch(evt Event, params ...Param) {
	if evt == EventNone {
		b.DispatchError(ErrInvalidEvent)
		return
	}
	dispatch := &busDispatch{
		event:  evt,
		params: params,
		future: syncx.SymbolicFuture[error](),
	}
	b.events <- dispatch
}

// DispatchResult will submit an event to the [EventBus] for propagation, and block until a result is returned.
// If an error is returned, then an [EventAsyncError] is still propagated to an appropriate handler, if registered.
func (b *EventBus) DispatchResult(evt Event, params ...Param) syncx.Future[error] {
	if evt == EventNone {
		b.DispatchError(ErrInvalidEvent)
		return syncx.StaticFuture(ErrInvalidEvent)
	}
	dispatch := &busDispatch{
		event:  evt,
		params: params,
		future: syncx.NewFuture[error](),
	}
	b.events <- dispatch
	return dispatch.future
}

func (b *EventBus) DispatchErrorf(format string, args ...any) {
	b.DispatchError(fmt.Errorf(format, args...))
}

func (b *EventBus) DispatchError(err error) {
	b.Dispatch(EventAsyncError, err)
}

func (b *EventBus) Register(id HandlerID, handledEvent Event, handler Handler) {
	syncx.LockFunc(&b.mux, func() {
		b.handlers[id] = handler
		b.handledEvents[handledEvent] = b.handledEvents[handledEvent].Add(id)
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
			return fmt.Errorf("expected a single error parameter: %w", err)
		}
		handler(err)
		return nil
	}))
}

func (b *EventBus) UnRegister(id HandlerID) {
	syncx.LockFunc(&b.mux, func() {
		handler, ok := b.handlers[id]
		if !ok {
			return
		}
		handler.Stop()
		delete(b.handlers, id)
	})
}

func (b *EventBus) AddHandledEvent(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		_, ok := b.handlers[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		b.handledEvents[evt] = b.handledEvents[evt].Add(id)
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
		return nil
	})
}

// Start will start processing of dispatched events if it's not started already.
// Once the [EventBus] has stopped processing events, it cannot be restarted.
// This is safe to call multiple times from multiple goroutines. Only the first call to start will begin processing.
func (b *EventBus) Start(ctx context.Context) *EventBus {
	b.dispatchLoop.Do(func() {
		b.doneDispatching.Add(b.numWorkers)
		// Must cache events channel so a goroutine doesn't block after a call to Stop.
		events := b.events
		for i := 0; i < b.numWorkers; i++ {
			go b.start(ctx, events)
		}
	})
	return b
}

func (b *EventBus) start(ctx context.Context, events chan *busDispatch) {
	defer b.doneDispatching.Done()
	defer func() {
		for _, handler := range b.handlers {
			handler.Stop()
		}
	}()
	var errs []error
	for {
		if len(errs) > 0 {
			// Dispatch errors
			syncx.RLockFunc(&b.mux, func() {
				errHandlerIDs := b.handledEvents[EventAsyncError]
				if len(errHandlerIDs) == 0 {
					// No registered error handlers, nothing to do.
					return
				}
				for _, err := range errs {
					for id := range errHandlerIDs {
						handler := b.handlers[id]
						if handler == nil {
							continue
						}
						// No recourse for error handler returning an error in this context.
						_ = handler.HandleEvent(EventAsyncError, err)
					}
				}
			})
			errs = nil
		}
		select {
		case <-ctx.Done():
			b.Stop()
			return
		case dispatch, more := <-events:
			if !more {
				return
			}
			syncx.RLockFunc(&b.mux, func() {
				defer func() {
					// If a result has already been returned or a result is not requested, then this does nothing
					dispatch.future.Resolve(nil)
				}()

				// Locate relevant handlers
				handlers := b.handledEvents[dispatch.event]
				noHandlersMessage := fmt.Errorf("%w for event %d", ErrNoHandler, dispatch.event)

				// None found
				if len(handlers) == 0 {
					// Check if this is already an EventAsyncError
					if dispatch.event != EventAsyncError {
						dispatch.future.Resolve(noHandlersMessage)
						errs = append(errs, noHandlersMessage)
					}
					return
				}

				// Dispatch to all relevant handlers
				for id := range handlers {
					handler := b.handlers[id]
					if handler == nil {
						continue
					}
					err := handler.HandleEvent(dispatch.event, dispatch.params...)
					if err != nil {
						// Return first error
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
	b.stopDispatch.Do(func() {
		events := b.events
		b.events = nil
		close(events)
	})
}

// AwaitStop will halt event processing for the [EventBus] if it's running, and wait for processing to stop.
// A timeout value may be used to set a deadline for stopping.
// Calling this when the [EventBus] is already stopped will return immediately.
func (b *EventBus) AwaitStop(timeout time.Duration) {
	b.Stop()
	wait, cancel := context.WithTimeout(context.Background(), timeout)
	go func() {
		defer cancel()
		b.doneDispatching.Wait()
	}()
	<-wait.Done()
}
