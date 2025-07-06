package iterx

import (
	"cmp"
	"math"
	"slices"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Max will return the max value in the given iterator.
func Max[T Number](iter SliceIter[T]) T {
	var _max T
	iter(func(val T) bool {
		_max = max(val, _max)
		return true
	})
	return _max
}

// Min will return the minimum value in the given iterator.
func Min[T Number](iter SliceIter[T]) T {
	var _min T
	iter(func(val T) bool {
		_min = min(val, _min)
		return true
	})
	return _min
}

// Sum will return the sum of all numbers in the SliceIter.
// This function does not check for over/underflow conditions.
func Sum[T Number](iter SliceIter[T]) float64 {
	var sum float64
	iter(func(val T) bool {
		sum += float64(val)
		return true
	})
	return sum
}

// Average will return the average of all numbers in the SliceIter.
// This function does not check for over/underflow conditions.
func Average[T Number](iter SliceIter[T]) float64 {
	var (
		sum, count float64
	)
	iter(func(val T) bool {
		sum += float64(val)
		count++
		return true
	})
	return sum / count
}

// StdDev calculates the standard deviation of a population as represented by the given SliceIter.
// This function does not check for over/underflow conditions.
func StdDev[T Number](iter SliceIter[T]) float64 {
	var (
		sumsq float64
		count float64
	)
	average := Average(iter)
	iter(func(val T) bool {
		diff := float64(val) - average
		sumsq += diff * diff
		count++
		return true
	})
	return math.Sqrt(sumsq / count)
}

func Sort[T cmp.Ordered](iter SliceIter[T]) SliceIter[T] {
	slice := iter.Slice()
	slices.Sort(slice)
	return Select(slice)
}

func ReverseSort[T cmp.Ordered](iter SliceIter[T]) SliceIter[T] {
	slice := iter.Slice()
	slices.Sort(slice)
	slices.Reverse(slice)
	return Select(slice)
}
