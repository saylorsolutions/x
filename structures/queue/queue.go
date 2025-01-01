package queue

import (
	"iter"
	"sync"
)

// Queue is a concurrency-safe queue implementation.
type Queue[T any] struct {
	mux    sync.RWMutex
	values []*queueElement[T]
}

func NewQueue[T any](initialBuffer ...int) *Queue[T] {
	if len(initialBuffer) > 0 {
		return &Queue[T]{values: make([]*queueElement[T], 0, initialBuffer[0])}
	}
	return &Queue[T]{}
}

type queueElement[T any] struct {
	val      T
	priority uint
}

// Len gets the length of the Queue
func (q *Queue[T]) Len() int {
	q.mux.RLock()
	defer q.mux.RUnlock()
	return len(q.values)
}

// PushRanked will insert an item in the Queue such that its priority is greater than all elements after it.
// If priority is set to zero, then the item will be appended to the tail.
func (q *Queue[T]) PushRanked(val T, priority uint) {
	q.mux.Lock()
	defer q.mux.Unlock()
	if priority == 0 {
		q.values = append(q.values, &queueElement[T]{val: val, priority: 0})
		return
	}
	var (
		insertPos int
		found     bool
	)
	for i, el := range q.values {
		if el.priority < priority {
			insertPos = i
			found = true
			break
		}
	}
	if !found {
		q.values = append(q.values, &queueElement[T]{val: val, priority: priority})
		return
	}
	q.values = append(q.values[:insertPos],
		append([]*queueElement[T]{{val: val, priority: priority}}, q.values[insertPos:]...)...)
}

func (q *Queue[T]) pushHead(el *queueElement[T]) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.values = append([]*queueElement[T]{el}, q.values...)
}

// Push will push an item to the tail of the Queue.
func (q *Queue[T]) Push(val T) {
	q.PushRanked(val, 0)
}

// Pop will pop an item from the head of the Queue.
// False will be returned if the Queue is empty.
func (q *Queue[T]) Pop() (T, bool) {
	element, ok := q.pop()
	if !ok {
		var mt T
		return mt, false
	}
	return element.val, true
}

func (q *Queue[T]) pop() (*queueElement[T], bool) {
	q.mux.Lock()
	defer q.mux.Unlock()
	if len(q.values) == 0 {
		return nil, false
	}
	element := q.values[0]
	q.values = q.values[1:]
	return element, true
}

func (q *Queue[T]) iterator() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			val, ok := q.Pop()
			if !ok {
				return
			}
			if !yield(val) {
				return
			}
		}
	}
}
