package syncx

import (
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
