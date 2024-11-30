package observer

import (
	"context"
	"github.com/saylorsolutions/x/syncx"
	"sync"
)

// Observer receives a new value from a [Subject] when it changes.
type Observer[T any] func(newVal T)

// Subject is a value that may be observed for changes.
type Subject[T any] interface {
	Get() T
	Set(newVal T)
	Observe(obs Observer[T])
}

// NewSubject creates a [Subject] implementation with a context for cancellation.
// Once the context is cancelled, the [Subject] will no longer propagate changes.
func NewSubject[T any](ctx context.Context, val T) Subject[T] {
	changes := make(chan T, 1)
	sub := &subject[T]{
		changes: changes,
		value:   val,
	}
	go processChanges[T](ctx, sub, changes)
	return sub
}

func processChanges[T any](ctx context.Context, sub *subject[T], changes chan T) {
	defer func() {
		sub.changes = nil
		close(changes)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case val, more := <-changes:
			if !more {
				return
			}
			syncx.LockFunc(&sub.mux, func() {
				sub.value = val
				// This needs to be in the same locking function so the value and observers don't change at the same time.
				for _, obs := range sub.observers {
					obs(val)
				}
			})
		}
	}
}

type subject[T any] struct {
	changes chan<- T

	mux       sync.RWMutex
	value     T
	observers []Observer[T]
}

func (s *subject[T]) Get() T {
	return syncx.RLockFuncT(&s.mux, func() T {
		return s.value
	})
}

func (s *subject[T]) Set(newVal T) {
	s.changes <- newVal
}

func (s *subject[T]) Observe(obs Observer[T]) {
	syncx.LockFunc(&s.mux, func() {
		s.observers = append(s.observers, obs)
	})
}
