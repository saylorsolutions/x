/*
Package eventbus provides an event bus implementation that allows loose coupling and event-based processing in an application.

# Design Priorities

Here are the design priorities of the implementation:

  - It should be as deterministic as possible by reducing opportunities for race conditions.
  - It should be transparent in its results by reporting errors that occur in event handlers back to dispatching code or specific handlers.
  - It should be simple in its operations to add as little complexity as possible, and abstract the user from the more complex aspects of asynchronous processing.
  - It should be made easy to integrate by providing helpers for the rough edges around transferring arbitrary data asynchronously.
  - It should require little in terms of constraints to make it more applicable to a wide variety of problem domains.

# EventBus Primitives

Every [Event] is an integer representing something specific that happens in an application.
It's recommended to create a global enum of [Event] that is accessible to all parts of the application to have a consistent, documented reference of events.

Note that there are two reserved event numbers: [EventNone] and [EventAsyncError] that are set to 0 and 1, respectively.
These event numbers should not be used with different semantics, as they're used internally.

An event may be accompanied by one or more [Param] that provide additional details for understanding the event.
A [Param] may be any type and this can introduce some complexity in understanding the parameter.
To address this, I added the concept of a [ParamAssertion], which is just a function that makes some assertion about a [Param].
Many [ParamAssertion] can be combined with [ParamSpec] to create a function that applies all [ParamAssertion] to all parameters.

See patterns/eventbus/paramspec_test.go for an example of this.

# EventBus Initialization

There are two distinct ways to initialize an [EventBus]:
  - Use the [Instance] function to get a global singleton [EventBus].
  - Use the [NewEventBus] function to get an instance.

In either case the global [DefaultBufferSize] variable is cached to determine the size of the events channel buffer.
The cached value is also used to determine how many goroutines should be created to handle dispatched events.

The [EventBus] will only dispatch an [Event] to a [Handler] if it has been previously registered to handle the specific [Event].
To register a [Handler], use [EventBus.Register] with a string handler ID, the event that the [Handler] should handle, and the [Handler] implementation.
To allow a [Handler] to handle multiple events, use [EventBus.AddHandledEvent] with the string handler ID, and the additional [Event] that should be dispatched to the [Handler].
Note that - for simpler event handling cases - a [HandlerFunc] may be used when the function doesn't need to be aware of the [EventBus] stopping, and doesn't need to free resources.

To receive and handle errors that occur while handling events, use [EventBus.RegisterErrorHandler] to register a function that is called for each error.
This can be useful for consolidating logging for errors that occur in a [Handler].

To start propagation of events, use [EventBus.Start] with a context.
When the context is cancelled, all event processing will stop after the [EventBus] has worked through all dispatched events.
To stop the [EventBus] and wait for processing to fully stop, use [EventBus.AwaitStop].
This method will block until all processing goroutines have stopped, or the timeout has been reached.

# Event Flow

Components of the application may use [EventBus.Dispatch] to emit events to the bus.
This event will be dispatched to all currently registered handlers that indicated they can handle the event.

To also receive an error from a [Handler], use [EventBus.DispatchResult] that returns a [Future].
The first error that occurs will be returned via the [Future], and all errors will still be dispatched to any registered error handlers.

To dispatch an error outside a [Handler], use either the [EventBus.DispatchError] or [EventBus.DispatchErrorf] methods.
Note that using these methods in handlers can cause a deadlock when the processing goroutines are trying to process events while handlers are trying to dispatch errors.
To errors in Handlers should just be returned from the processing method/function.

[Future]: github.com/saylorsolutions/x/syncx/future.go
*/
package eventbus
