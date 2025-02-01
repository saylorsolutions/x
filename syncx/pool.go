package syncx

import "sync"

// Pool provides a generic wrapper of [sync.Pool].
type Pool[T any] struct {
	pool sync.Pool
}

// NewPool is used to create a typed [Pool].
func NewPool[T any](factory func() T) *Pool[T] {
	if factory == nil {
		panic("nil factory function")
	}
	p := new(Pool[T])
	p.pool.New = func() any {
		return factory()
	}
	return p
}

// Get selects an arbitrary item from the [Pool], removes it from the [Pool], and returns it to the caller.
// Callers should not assume any relation between values passed to [Pool.Put] and the values returned by Get.
//
// If the [Pool] is empty, then Get returns the result of calling the provided factory function.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put adds val to the [Pool].
func (p *Pool[T]) Put(val T) {
	p.pool.Put(val)
}
