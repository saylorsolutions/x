package queue

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestNewChannelQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cq, err := NewChannelQueue[int](ctx, OptChannelSize(1), OptInitialBuffer(10))
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

func TestChannelSize(t *testing.T) {
	tests := []struct {
		Val     int
		IsError bool
	}{
		{Val: -1, IsError: true},
		{Val: 0},
		{Val: 1},
	}

	conf := new(channelQueueConfig)
	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			conf.channelSize = 0
			err := OptChannelSize(test.Val)(conf)
			if test.IsError {
				assert.Equal(t, 0, conf.channelSize)
				assert.Error(t, err)
			} else {
				assert.Equal(t, conf.channelSize, test.Val)
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitialBuffer(t *testing.T) {
	tests := []struct {
		Val     int
		IsError bool
	}{
		{Val: -1, IsError: true},
		{Val: 0},
		{Val: 1},
	}

	conf := new(channelQueueConfig)
	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			conf.queueInitialBuffer = 0
			err := OptInitialBuffer(test.Val)(conf)
			if test.IsError {
				assert.Equal(t, 0, conf.queueInitialBuffer)
				assert.Error(t, err)
			} else {
				assert.Equal(t, conf.queueInitialBuffer, test.Val)
				assert.NoError(t, err)
			}
		})
	}
}
