package eventbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/syncx"
	"sync"
	"time"
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
	// DispatchBuffer dictates the size of the dispatch channel.
	// May be increased to optimize for higher throughput and less blocking at the expense of more reserved memory.
	DispatchBuffer = 1
)

var (
	instanceBus *EventBus
	initOnce    sync.Once
)

// Instance is useful in cases where a single, global [EventBus] is desired.
// This can be helpful for accessing and synchronizing events.
// The global visibility means it's likely not a good fit for concurrent access, since it introduces the potential for configuration race conditions.
func Instance() *EventBus {
	initOnce.Do(func() {
		instanceBus = NewEventBus()
	})
	return instanceBus
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers:  map[HandlerID]EventHandler{},
		eventMask: map[HandlerID]map[Event]bool{},
		events:    make(chan *busDispatch, DispatchBuffer),
	}
}

type Param any

func Paramf(format string, args ...any) Param {
	return Param(fmt.Sprintf(format, args...))
}

type HandlerID string

type EventHandler interface {
	// HandleEvent will handle the given event and do some kind of processing.
	// Returned errors will be reported with a dispatched [EventAsyncError].
	HandleEvent(evt Event, params ...Param) error
	// Stop will alert the [EventHandler] that it should clean up resources and reject further events.
	// This can be ignored if not needed.
	Stop()
}

// EventHandlerFunc is a function that implements the EventHandler interface.
// This is intended for simple event handling cases where [EventHandler.Stop] has no real semantics for the handling component.
type EventHandlerFunc func(evt Event, params ...Param) error

func (f EventHandlerFunc) HandleEvent(evt Event, params ...Param) error {
	return f(evt, params...)
}

func (f EventHandlerFunc) Stop() {}

type busDispatch struct {
	event  Event
	params []Param
	future syncx.Future[error]
}

type EventBus struct {
	events          chan *busDispatch
	dispatchLoop    sync.Once
	stopDispatch    sync.Once
	doneDispatching sync.WaitGroup

	mux       sync.RWMutex
	handlers  map[HandlerID]EventHandler
	eventMask map[HandlerID]map[Event]bool
}

// Dispatch will submit an event to the [EventBus] for propagation.
// If an error occurs, then an [EventAsyncError] is propagated to an appropriate handler, if registered.
// If the EventBus is stopping, then this call will block indefinitely.
func (b *EventBus) Dispatch(evt Event, params ...Param) {
	if evt == EventNone {
		b.DispatchError("attempted call to Dispatch with missing Event code: %#v", params)
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
		b.DispatchError("attempted call to Dispatch with missing Event code: %#v", params)
		return syncx.StaticFuture(fmt.Errorf("attempted call to Dispatch with missing Event code: %#v", params))
	}
	dispatch := &busDispatch{
		event:  evt,
		params: params,
		future: syncx.NewFuture[error](),
	}
	b.events <- dispatch
	return dispatch.future
}

func (b *EventBus) DispatchError(format string, args ...any) {
	err := fmt.Sprintf(format, args...)
	b.Dispatch(EventAsyncError, err)
}

func (b *EventBus) Register(id HandlerID, handler EventHandler, handledEvents ...Event) {
	syncx.LockFunc(&b.mux, func() {
		b.handlers[id] = handler
		b.eventMask[id] = map[Event]bool{}
		for _, evt := range handledEvents {
			b.eventMask[id][evt] = true
		}
	})
}

func (b *EventBus) RegisterErrorHandler(id HandlerID, handler func(error)) {
	b.Register(id, EventHandlerFunc(func(evt Event, params ...Param) error {
		var (
			err  error
			serr string
		)
		spec := ParamSpec(1,
			AnyPass(
				AssertAndStore(&err),
				AssertAndStore(&serr),
			),
		)
		if errs := spec(params); len(errs) > 0 {
			return fmt.Errorf("expected a single error/string parameter: %w", err)
		}
		if len(serr) > 0 {
			err = errors.New(serr)
		}
		handler(err)
		return nil
	}), EventAsyncError)
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
		events, ok := b.eventMask[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		events[evt] = true
		return nil
	})
}

func (b *EventBus) AddHandledExclusive(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		events, ok := b.eventMask[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		for curID, handledEvents := range b.eventMask {
			if curID == id {
				continue
			}
			if handledEvents == nil {
				continue
			}
			delete(handledEvents, evt)
		}
		events[evt] = true
		return nil
	})
}

func (b *EventBus) RemoveHandledEvent(id HandlerID, evt Event) error {
	return syncx.LockFuncT(&b.mux, func() error {
		events, ok := b.eventMask[id]
		if !ok {
			return fmt.Errorf("no registered handler with id '%s'", id)
		}
		delete(events, evt)
		return nil
	})
}

// Start will start processing of dispatched events if it's not started already.
// Once the [EventBus] has stopped processing events, it cannot be restarted.
// This is safe to call multiple times from multiple goroutines. Only the first call to start will begin processing.
func (b *EventBus) Start(ctx context.Context) {
	b.dispatchLoop.Do(func() {
		b.doneDispatching.Add(1)
		go func() {
			defer b.doneDispatching.Done()
			defer func() {
				for _, handler := range b.handlers {
					handler.Stop()
				}
			}()
			for {
				select {
				case <-ctx.Done():
					b.Stop()
					return
				case dispatch, more := <-b.events:
					if !more {
						return
					}
					syncx.RLockFunc(&b.mux, func() {
						defer func() {
							// If a result has already been returned or a result is not requested, then this does nothing
							dispatch.future.Resolve(nil)
						}()

						// Locate relevant handlers
						handlers := map[HandlerID]struct{}{}
						for id, handles := range b.eventMask {
							if handles == nil {
								continue
							}
							if handles[dispatch.event] {
								handlers[id] = struct{}{}
							}
						}
						noHandlersMessage := fmt.Sprintf("No handlers for event %d", dispatch.event)
						if len(handlers) == 0 {
							// None found
							if dispatch.event != EventAsyncError {
								dispatch.future.Resolve(errors.New(noHandlersMessage))
								b.DispatchError(noHandlersMessage)
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
								b.DispatchError("Handler '%s' failed to handle event %d: %v", id, dispatch.event, err)
							}
						}
					})
				}
			}
		}()
	})
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
