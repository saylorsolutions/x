package queue

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue[int]()
	assert.Equal(t, 0, q.Len())
	val, ok := q.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	q.Push(1)
	assert.Equal(t, 1, q.Len())

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	q.PushRanked(3, 0)
	assert.Equal(t, 1, q.Len())
	q.PushRanked(1, 2)
	assert.Equal(t, 2, q.Len())
	q.PushRanked(2, 1)
	assert.Equal(t, 3, q.Len())

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 1, val)
	assert.Equal(t, 2, q.Len())
	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 2, val)
	assert.Equal(t, 1, q.Len())
	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 3, val)
	assert.Equal(t, 0, q.Len())
}
