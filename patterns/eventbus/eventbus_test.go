package eventbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testEvent           Event = 5
	testNotHandledEvent       = 99
	testShutdownTimeout       = time.Second
	testAwaitTimeout          = 100 * time.Millisecond
)

func TestEventBus_Dispatch(t *testing.T) {
	var (
		errorReceived, handlerCalled bool
	)
	bus := testEventBus(&handlerCalled, &errorReceived)
	defer bus.AwaitStop(testShutdownTimeout)

	bus.Dispatch(testEvent, "A message")
	time.Sleep(50 * time.Millisecond)

	assert.False(t, errorReceived, "Should not receive an error")
	assert.True(t, handlerCalled, "Handler should have been called")
}

func TestEventBus_Dispatch_MissingEvent(t *testing.T) {
	bus := NewEventBus().Start(context.Background())
	defer bus.AwaitStop(testShutdownTimeout)
	err := bus.DispatchResult(testNotHandledEvent).Await(testAwaitTimeout)
	assert.ErrorIs(t, err, ErrNoHandler, "Should be rejected because there's no handler")

	err = bus.DispatchResult(EventNone, "A message").Await(testAwaitTimeout)
	assert.ErrorIs(t, err, ErrInvalidEvent, "Should be rejected because an invalid event is used")
}

func TestEventBus_DispatchResult(t *testing.T) {
	var (
		errorReceived, handlerCalled bool
	)
	bus := testEventBus(&handlerCalled, &errorReceived)
	defer bus.AwaitStop(testShutdownTimeout)

	if err := bus.DispatchResult(testEvent, "A message").Await(testAwaitTimeout); err != nil {
		t.Errorf("Should not have received error: %v", err)
	}
	assert.False(t, errorReceived, "Should not receive an error")
	assert.True(t, handlerCalled, "Handler should have been called")

	handlerCalled = false
	errorReceived = false
	if err := bus.DispatchResult(testEvent, 5).Await(testAwaitTimeout); err == nil {
		t.Error("Should have received error")
	}
	assert.True(t, errorReceived, "Should receive an error")
	assert.True(t, handlerCalled, "Handler should have been called")
}

func TestEventBus_Dispatch_Async(t *testing.T) {
	old := DefaultBufferSize
	defer func() {
		DefaultBufferSize = old
	}()
	DefaultBufferSize = 3
	var (
		counter   atomic.Int32
		asyncErrs atomic.Int32
	)

	bus := testEventBus(nil, nil)
	bus.Register("counter", testEvent, HandlerFunc(func(evt Event, params ...Param) error {
		counter.Add(1)
		return nil
	}))
	assert.NoError(t, bus.AddHandledExclusive("counter", testEvent))
	bus.RegisterErrorHandler("err-handler", func(err error) {
		asyncErrs.Add(1)
		t.Errorf("Should not have received an error: %v", err)
	})

	for i := 0; i < 3; i++ {
		bus.Dispatch(testEvent, fmt.Sprintf("%d", i))
	}
	bus.AwaitStop(testShutdownTimeout)
	assert.Equal(t, int32(3), counter.Load())
	assert.Equal(t, int32(0), asyncErrs.Load())
}

func TestDispatchBuffer_Invalid(t *testing.T) {
	old := DefaultBufferSize
	defer func() {
		DefaultBufferSize = old
	}()
	DefaultBufferSize = 0
	assert.Panics(t, func() {
		testEventBus(nil, nil)
	})
	DefaultBufferSize = old
	buf := NewEventBus()
	DefaultBufferSize = 0
	assert.NotPanics(t, func() {
		buf.Start(context.Background())
	})
	assert.Equal(t, old, buf.eventsSize)
	buf.AwaitStop(testShutdownTimeout)
}

func TestEventBus_Stop(t *testing.T) {
	bus := NewEventBus()
	handler := new(testHandlerImpl)
	bus.Register("stopping-handler", testEvent, handler)
	bus.Start(context.Background())

	for i := 0; i < 3; i++ {
		bus.Dispatch(testEvent, fmt.Sprintf("%d", i))
	}
	bus.AwaitStop(testShutdownTimeout)
	assert.Equal(t, 3, handler.count)
	assert.Equal(t, 1, handler.stoppedCount)
	assert.True(t, handler.stopped)
}

var _ Handler = (*testHandlerImpl)(nil)

// This isn't really representative of a good [Handler].
// In reality, we should probably have some locking over these fields to prevent race conditions.
type testHandlerImpl struct {
	count        int
	stoppedCount int
	stopped      bool
}

func (t *testHandlerImpl) HandleEvent(evt Event, params ...Param) error {
	t.count++
	return nil
}

func (t *testHandlerImpl) Stop() {
	t.stoppedCount++
	t.stopped = true
}

func testEventBus(handlerCalled, errorReceived *bool) *EventBus {
	bus := NewEventBus().Start(context.Background())
	bus.Register("test-handler", testEvent, HandlerFunc(func(evt Event, params ...Param) error {
		if handlerCalled != nil {
			*handlerCalled = true
		}
		var param string
		spec := ParamSpec(1,
			AssertAndStore(&param),
		)
		if errs := spec(params); len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	}))
	if errorReceived != nil {
		bus.RegisterErrorHandler("error-handler", testShouldNotFail(errorReceived))
	}
	return bus
}

func testShouldNotFail(received *bool) func(err error) {
	return func(err error) {
		*received = true
	}
}
