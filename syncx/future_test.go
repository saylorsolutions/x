package syncx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFuture_Await(t *testing.T) {
	var order = make([]int, 0, 4)
	f := NewFuture[int]()
	order = append(order, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		order = append(order, 2)
		f.Resolve(3)

		// Make sure that subsequent calls don't actually do anything
		f.Resolve(5)
		f.Resolve(6)
		f.Resolve(7)
	}()
	order = append(order, f.Await())
	assert.Equal(t, 3, f.Await(), "The same value should be returned again with Await")
	order = append(order, 4)
	assert.Equal(t, []int{1, 2, 3, 4}, order, "Processing should happen in the expected order")
}

func TestFuture_Await_Blocking(t *testing.T) {
	var (
		f       = NewFutureErr[int]()
		process = func(f FutureErr[int]) {
			time.Sleep(150 * time.Millisecond)
			f.ResolveErr(5, nil)
		}
	)

	go process(f)
	for i := 0; i < 3; i++ {
		switch i {
		case 0:
			fallthrough
		case 1:
			fallthrough
		case 2:
			val, err := f.AwaitErr(40 * time.Millisecond)
			assert.Equal(t, 0, val)
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		}
	}
	val, err := f.AwaitErr()
	assert.Equal(t, 5, val)
	assert.NoError(t, err)
}
