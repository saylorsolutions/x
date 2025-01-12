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
	set = New("a", "b")
	set.Add("c", "d")
	set.Remove("d", "e")
	assert.True(t, set.HasAll("a", "b", "c"))
	assert.Empty(t, set.Difference(New("a", "b", "c")))

	slice = set.Slice()
	assert.Len(t, slice, 3)
	sort.Strings(slice)
	assert.Equal(t, "a", slice[0])
	assert.Equal(t, "b", slice[1])
	assert.Equal(t, "c", slice[2])
}

func TestSet_Slice_NilSet(t *testing.T) {
	var set Set[int]
	assert.Nil(t, set.Slice())
	assert.Empty(t, set.Slice())
	assert.Nil(t, set.Copy().Slice())
	assert.Empty(t, set.Copy().Slice())
}
