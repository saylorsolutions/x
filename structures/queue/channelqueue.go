package queue

import (
	"context"
	"fmt"
	"sync"
)

// ChannelQueue is used to create a [Queue] that can be consumed as a channel.
// It creates a worker goroutine to manage sending and receiving.
//
// This is good for cases like:
//   - When an arbitrary sized queue is needed, but a channel is more convenient.
//   - Where a dynamically buffered channel is desired to prevent deadlocking on use.
//   - When it's not known how many consumers/producers will be used ahead of use.
type ChannelQueue[T any] struct {
	// C is the channel where queue values will be posted.
	C       <-chan T
	queue   *Queue[T]
	ctx     context.Context
	stop    context.CancelFunc
	recv    chan *queueElement[T]
	disp    chan T
	doStop  sync.Once
	stopped chan struct{}
}

type channelQueueConfig struct {
	queueInitialBuffer int
	channelSize        int
}

type ChannelQueueOption func(conf *channelQueueConfig) error

// ChannelSize is used to set the buffer size of the input and output channels.
func ChannelSize(size int) ChannelQueueOption {
	return func(conf *channelQueueConfig) error {
		if size < 0 {
			return fmt.Errorf("invalid channel size '%d'", size)
		}
		conf.channelSize = size
		return nil
	}
}

// InitialBuffer is used to set the initial size of the internal [Queue].
func InitialBuffer(size int) ChannelQueueOption {
	return func(conf *channelQueueConfig) error {
		if size < 0 {
			return fmt.Errorf("invalid queue initial buffer size '%d'", size)
		}
		conf.queueInitialBuffer = size
		return nil
	}
}

// NewChannelQueue creates a new [ChannelQueue], and starts a goroutine to keep data flowing.
func NewChannelQueue[T any](ctx context.Context, opts ...ChannelQueueOption) (*ChannelQueue[T], error) {
	conf := new(channelQueueConfig)
	for _, opt := range opts {
		if err := opt(conf); err != nil {
			return nil, err
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var (
		cancel context.CancelFunc
	)
	ctx, cancel = context.WithCancel(ctx)
	cq := &ChannelQueue[T]{
		ctx:     ctx,
		stop:    cancel,
		stopped: make(chan struct{}),
	}

	if conf.queueInitialBuffer > 0 {
		cq.queue = NewQueue[T](conf.queueInitialBuffer)
	} else {
		cq.queue = NewQueue[T]()
	}

	if conf.channelSize == 0 {
		cq.recv = make(chan *queueElement[T])
		cq.disp = make(chan T)
	} else {
		cq.recv = make(chan *queueElement[T], conf.channelSize)
		cq.disp = make(chan T, conf.channelSize)
	}
	cq.C = cq.disp
	go cq.worker()
	return cq, nil
}

func (q *ChannelQueue[T]) worker() {
	defer close(q.stopped)
	defer close(q.disp)
	var (
		stopping bool
		head     *queueElement[T]
		haveHead bool
	)

	for {
		if stopping {
			// Stopping, drain recv and push all remaining in queue.
			for {
				select {
				case _new := <-q.recv:
					q.queue.PushRanked(_new.val, _new.priority)
					continue
				default:
					// No more goroutines waiting on sending, close receiver.
					recv := q.recv
					q.recv = nil
					close(recv)
				}
				break
			}
			iter := q.queue.iterator()
			iter(func(val T) bool {
				q.disp <- val
				return true
			})
			return
		}
		head, haveHead = q.queue.pop()
		switch {
		case haveHead:
			// Not empty, listen to both.
			select {
			case _new := <-q.recv:
				q.queue.pushHead(head)
				q.queue.PushRanked(_new.val, _new.priority)
			case q.disp <- head.val:
				// Dispatched, loop around for next element.
			case <-q.ctx.Done():
				q.queue.pushHead(head)
				stopping = true
				q.Stop()
			}
		default:
			// Empty, wait for push.
			select {
			case _new := <-q.recv:
				q.queue.PushRanked(_new.val, _new.priority)
			case <-q.ctx.Done():
				stopping = true
				q.Stop()
			}
		}
	}
}

// Stop will signal that the goroutine managing the ChannelQueue should clean up and stop operating.
// This is implicitly called when the given context is cancelled.
func (q *ChannelQueue[T]) Stop() {
	q.doStop.Do(func() {
		q.stop()
	})
}

// AwaitStop will call [ChannelQueue.Stop] and wait for all operations to cease before returning.
func (q *ChannelQueue[T]) AwaitStop() {
	q.Stop()
	q.Await()
}

// Await will wait for all [ChannelQueue] operations to cease before returning.
func (q *ChannelQueue[T]) Await() {
	<-q.stopped
}

// Len gets the length of the Queue
func (q *ChannelQueue[T]) Len() int {
	return q.queue.Len()
}

// PushRanked will insert an item in the Queue such that its priority is greater than all elements after it.
// If priority is set to zero, then the item will be appended to the tail.
func (q *ChannelQueue[T]) PushRanked(val T, priority uint) {
	select {
	case <-q.ctx.Done():
		return
	default:
		q.recv <- &queueElement[T]{val: val, priority: priority}
	}
}

// Push will push an item to the tail of the ChannelQueue.
func (q *ChannelQueue[T]) Push(val T) {
	q.PushRanked(val, 0)
}

// Pop will pop an item from the head of the ChannelQueue.
// False will be returned if the ChannelQueue is empty.
func (q *ChannelQueue[T]) Pop() (T, bool) {
	select {
	case val, more := <-q.disp:
		if !more {
			var mt T
			return mt, false
		}
		return val, true
	default:
		var mt T
		return mt, false
	}
}
