package iterx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInvertMap(t *testing.T) {
	initial := map[int]string{
		0: "0",
		1: "1",
		2: "2",
	}
	result := InvertMap(SelectMap(initial)).Map()
	expected := map[string][]int{
		"0": {0},
		"1": {1},
		"2": {2},
	}
	assert.Equal(t, expected, result)
}

func TestSliceInverseMap(t *testing.T) {
	initial := []string{"a", "b", "c", "a", "b", "c", "d"}
	expected := map[string]int{
		"a": 0,
		"b": 1,
		"c": 2,
		"d": 6,
	}
	assert.Equal(t, expected, SliceInverseMap(initial).Map())
}

func TestSliceMap(t *testing.T) {
	initial := []int{4, 5, 6}
	expected := map[int]int{
		0: 4,
		1: 5,
		2: 6,
	}
	assert.Equal(t, expected, SliceMap(initial).Map())
}

func TestSliceSet(t *testing.T) {
	initial := []int{3, 4, 5, 5, 4, 3, 1}
	expected := map[int]bool{
		1: true,
		3: true,
		4: true,
		5: true,
	}
	assert.Equal(t, expected, SliceSet(initial).Map())
}

func TestMapIter_FilterKeysValues(t *testing.T) {
	initial := map[int]string{
		1: "1",
		2: "2",
		3: "3",
	}
	assert.Equal(t, map[int]string{1: "1"}, SelectMap(initial).
		FilterKeys(Equal(1)).
		Map(),
	)
	assert.Equal(t, map[int]string{2: "2"}, SelectMap(initial).
		FilterValues(Equal("2")).
		Map(),
	)
}

func TestMapIter_KeysValues(t *testing.T) {
	sliceIter := SliceMap([]int{1, 2, 3})
	assert.Equal(t, []int{0, 1, 2}, sliceIter.Keys().Slice())
	assert.Equal(t, []int{1, 2, 3}, sliceIter.Values().Slice())

	mapIter := SelectMap(map[int]int{
		0: 1,
		1: 2,
		2: 3,
	})
	sorted := mapIter.KeyOrder(Sort[int])
	assert.Equal(t, []int{0, 1, 2}, sorted.Keys().Slice())
	assert.Equal(t, []int{1, 2, 3}, sorted.Values().Slice())
	revSorted := mapIter.KeyOrder(ReverseSort[int])
	assert.Equal(t, []int{2, 1, 0}, revSorted.Keys().Slice())
	assert.Equal(t, []int{3, 2, 1}, revSorted.Values().Slice())
	revSorted = revSorted.AppendEntry(3, 4)
	assert.Equal(t, []int{2, 1, 0, 3}, revSorted.Keys().Slice())
	assert.Equal(t, []int{3, 2, 1, 4}, revSorted.Values().Slice())
	resorted := revSorted.KeyOrder(ReverseSort[int])
	assert.Equal(t, []int{3, 2, 1, 0}, resorted.Keys().Slice())
	assert.Equal(t, []int{4, 3, 2, 1}, resorted.Values().Slice())
}
