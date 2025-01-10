package bidimap

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	m := New[int, int]()
	assert.NotNil(t, m)
	assert.NotNil(t, m.ktov)
	assert.NotNil(t, m.vtok)
}

func TestBidiMap_Add(t *testing.T) {
	m := new(BidiMap[int, string])
	m.Add(1, "one")
	assert.True(t, m.HasValue("one"))
	assert.True(t, m.HasKey(1))
	assert.Equal(t, "one", m.Value(1))
	assert.Equal(t, 1, m.Key("one"))
}

func TestBidiMap_Concurrency(t *testing.T) {
	m := new(BidiMap[int, string])

	keys := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	values := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	var wg sync.WaitGroup
	wg.Add(len(keys))

	for i := 0; i < len(keys); i++ {
		go func(index int) {
			m.Add(keys[index], values[index])
			go func() {
				defer wg.Done()
				assert.True(t, m.HasKey(keys[index]))
				assert.True(t, m.HasValue(values[index]))
			}()
		}(i)
	}

	wg.Wait()
}
