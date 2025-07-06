package iterx

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestDedupe(t *testing.T) {
	slice := []int{0, 1, 0, 1, 2}
	expectedSliceDedupe := map[int]int{
		0: 0,
		1: 1,
		2: 2,
	}
	expectedMapDedupe := map[int]int{
		0: 0,
		1: 1,
		4: 2,
	}
	assert.Equal(t, expectedSliceDedupe,
		DedupeSlice(Select(slice)).WithIndex().Map())
	assert.Equal(t, expectedSliceDedupe,
		DedupeSlice(SliceMap(slice).Values()).WithIndex().Map())
	assert.Equal(t, expectedMapDedupe,
		DedupeValues(Select(slice).WithIndex()).Map())
	assert.Equal(t, expectedMapDedupe,
		DedupeValues(SliceMap(slice)).Map())
}

func TestSliceIter_Append(t *testing.T) {
	slice := SelectValue(5)
	slice = slice.AppendValue(10)
	slice = SelectValue(0).Append(slice)
	assert.Equal(t, []int{0, 5, 10}, slice.Slice())
}

func mustGet[T any](val T, ok bool) T {
	if !ok {
		panic("Expected to return a value")
	}
	return val
}

func TestSliceIter_OffsetLimit(t *testing.T) {
	slice := Select([]int{1, 2, 3, 4, 5})
	middle := slice.Offset(1).Limit(3)
	assert.Equal(t, 3, middle.Count())
	assert.Equal(t, 2, mustGet(middle.First()))
	assert.Equal(t, 4, mustGet(middle.Last()))
}

func TestSliceIter_Count(t *testing.T) {
	slice := Select([]int{1, 2, 3, 4, 5})
	assert.Equal(t, 5, slice.Count())
	assert.Equal(t, 3, slice.Offset(1).Limit(3).Count())
}

func TestTransformSlice(t *testing.T) {
	initial := []int{1, 2, 3}
	expected := []string{"1", "2", "3"}
	assert.Equal(t, initial, Select(initial).Slice())
	assert.Equal(t, expected, TransformSlice(Select(initial), func(in int) string {
		return strconv.Itoa(in)
	}).Slice())
}
