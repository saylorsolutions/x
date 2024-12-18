package set

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestSet_Slice(t *testing.T) {
	set := New[string]()
	slice := set.Slice()
	assert.Nil(t, slice)
	assert.Len(t, slice, 0)
	set = New("a", "b", "c")
	slice = set.Slice()
	assert.Len(t, slice, 3)
	sort.Strings(slice)
	assert.Equal(t, "a", slice[0])
	assert.Equal(t, "b", slice[1])
	assert.Equal(t, "c", slice[2])
}
