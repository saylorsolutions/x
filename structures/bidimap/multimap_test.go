package bidimap

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiMap_AddValues(t *testing.T) {
	m := new(MultiMap[int, string])
	assert.Nil(t, m.GetValues(1))
	assert.Empty(t, m.GetValues(1))

	m.AddValues(1, "a", "b", "c")
	values := m.GetValues(1)
	assert.NotNil(t, values)
	assert.Len(t, values, 3)
	assert.Contains(t, values, "a")
	assert.Contains(t, values, "b")
	assert.Contains(t, values, "c")

	for _, value := range values {
		keys := m.GetKeys(value)
		assert.Len(t, keys, 1)
		assert.Equal(t, 1, keys[0])
	}
}

func TestMultiMap_AddKeys(t *testing.T) {
	m := new(MultiMap[int, string])
	assert.Nil(t, m.GetKeys("1"))
	assert.Empty(t, m.GetKeys("1"))

	m.AddKeys("1", 1, 2, 3)
	keys := m.GetKeys("1")
	assert.NotNil(t, keys)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, 1)
	assert.Contains(t, keys, 2)
	assert.Contains(t, keys, 3)

	for _, key := range keys {
		values := m.GetValues(key)
		assert.Len(t, values, 1)
		assert.Equal(t, "1", values[0])
	}
}
