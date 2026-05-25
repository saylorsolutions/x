package observer

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSubject_Set(t *testing.T) {
	sub := NewSubject(t.Context(), 5)
	assert.Equal(t, 5, sub.Get())
	sub.Set(10)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 10, sub.Get())
}

func TestSubject_Observe(t *testing.T) {
	sub := NewSubject(t.Context(), 5)
	assert.Equal(t, 5, sub.Get())

	var (
		wg          sync.WaitGroup
		receivedVal int
	)
	wg.Add(1)
	sub.Observe(func(newVal int) {
		defer wg.Done()
		receivedVal = newVal
	})
	sub.Set(10)
	wg.Wait()
	assert.Equal(t, 10, receivedVal)

	wg.Add(1)
	sub.Set(15)
	wg.Wait()
	assert.Equal(t, 15, receivedVal)
}
