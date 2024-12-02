package syncx

import (
	"context"
	"sync"
	"time"
)

// Future is a value that is resolved asynchronously at a later time.
// Once Await returns, the value is cached for other calls to Await.
type Future[T any] interface {
	// Resolve sets the value of the [Future] so it can be resolved by consumers.
	// Only the first call to Resolve will set the result. Subsequent calls do nothing.
	Resolve(T)
	// Await blocks until the value is made available with [Future.Resolve], or until the timeout elapses if specified.
	// If the timeout limit is reached, then the [Future] type's zero value is returned.
	// If no timeout is given, then the function will wait indefinitely.
	Await(...time.Duration) T
}

func NewFuture[T any]() Future[T] {
	f := &future[T]{
		ch: make(chan *resultPair[T], 1),
	}
	f.cacheSet.Add(1)
	return f
}

// SymbolicFuture returns a [Future] that does nothing.
// It's used to satisfy an interface constraint when no actual result will be returned, and conserves resource.
func SymbolicFuture[T any]() Future[T] {
	return &staticFuture[T]{}
}

// StaticFuture returns the given value in response to [Future.Await], and [Future.Resolve] has no effect.
func StaticFuture[T any](val T) Future[T] {
	return &staticFuture[T]{
		staticVal: val,
	}
}

// FutureErr is the same as [Future], but it returns a value and an error.
type FutureErr[T any] interface {
	// ResolveErr sets the value (and possibly an error) of the [Future] so it can be resolved by consumers.
	// Only the first call to ResolveErr will set the result. Subsequent calls do nothing.
	ResolveErr(T, error)
	// AwaitErr blocks until the value is made available with [Future.Resolve], or until the timeout elapses if specified.
	// If the timeout limit is reached, then the [Future] type's zero value is returned along with the error returned from the context being cancelled.
	// If no timeout is given, then the function will wait indefinitely.
	AwaitErr(...time.Duration) (T, error)
}

func NewFutureErr[T any]() FutureErr[T] {
	f := &future[T]{
		ch: make(chan *resultPair[T], 1),
	}
	f.cacheSet.Add(1)
	return f
}

// SymbolicFutureErr returns a [FutureErr] that does nothing. It's used to satisfy an interface constraint when no actual result will be returned, and conserves resource.
func SymbolicFutureErr[T any]() FutureErr[T] {
	return &staticFuture[T]{}
}

// StaticFutureErr returns the given value in response to [FutureErr.AwaitErr], and [FutureErr.ResolveErr] has no effect.
func StaticFutureErr[T any](val T, err error) FutureErr[T] {
	return &staticFuture[T]{
		staticVal: val,
		err:       err,
	}
}

type resultPair[T any] struct {
	val T
	err error
}

type future[T any] struct {
	ch       chan *resultPair[T]
	resolve  sync.Once
	cacheVal T
	cacheErr error
	cacheSet sync.WaitGroup
}

func (f *future[T]) Resolve(val T) {
	f.ResolveErr(val, nil)
}

func (f *future[T]) Await(timeout ...time.Duration) T {
	val, _ := f.AwaitErr(timeout...)
	return val
}

func (f *future[T]) ResolveErr(val T, err error) {
	f.resolve.Do(func() {
		f.ch <- &resultPair[T]{val: val, err: err}
		close(f.ch)
		f.cacheVal = val
		f.cacheErr = err
		f.cacheSet.Done()
	})
}

func (f *future[T]) AwaitErr(timeout ...time.Duration) (T, error) {
	var (
		ctx    = context.Background()
		cancel = func() {}
	)
	if len(timeout) > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout[0])
	}
	defer cancel()
	return f.await(ctx)
}

func (f *future[T]) await(ctx context.Context) (T, error) {
	select {
	case pair, more := <-f.ch:
		if !more {
			f.cacheSet.Wait()
			return f.cacheVal, f.cacheErr
		}
		return pair.val, pair.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

type staticFuture[T any] struct {
	staticVal T
	err       error
}

func (n *staticFuture[T]) Resolve(T) {
}

func (n *staticFuture[T]) Await(...time.Duration) T {
	return n.staticVal
}

func (n *staticFuture[T]) ResolveErr(T, error) {
}

func (n *staticFuture[T]) AwaitErr(...time.Duration) (T, error) {
	return n.staticVal, n.err
}
