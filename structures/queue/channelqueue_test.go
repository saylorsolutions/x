package queue

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestNewChannelQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cq, err := NewChannelQueue[int](ctx, ChannelSize(1), InitialBuffer(10))
	require.NoError(t, err)

	var (
		sum      int
		expected int
		wg       sync.WaitGroup
		start    = time.Now()
	)
	wg.Add(10)

	for i := 0; i < 10; i++ {
		expected += i + 1
		go func() {
			defer wg.Done()
			cq.Push(i + 1)
		}()
	}
	go func() {
		defer cancel()
		for val := range cq.C {
			sum += val
		}
	}()
	wg.Wait()
	cq.AwaitStop()
	t.Log("Duration:", time.Since(start))
	assert.Equal(t, 0, cq.Len())
	val, ok := cq.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, val)
	<-ctx.Done()
	assert.Equal(t, expected, sum)
}
