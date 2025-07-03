package iterx

import "iter"

type SliceIter[T any] iter.Seq[T]

// SelectAll will translate any slice into a [SliceIter].
func SelectAll[T any](slice []T) SliceIter[T] {
	return Select(slice, Any[T]())
}

// Select will use the provided [Filter] to select elements from a slice, returning a [SliceIter].
func Select[T any](slice []T, filter Filter[T]) SliceIter[T] {
	return func(yield func(T) bool) {
		if filter == nil {
			panic("nil filter")
		}
		for _, element := range slice {
			if filter(element) {
				if !yield(element) {
					return
				}
			}
		}
	}
}

func (i SliceIter[T]) Slice() []T {
	var elements []T
	i(func(element T) bool {
		elements = append(elements, element)
		return true
	})
	return elements
}

func (i SliceIter[T]) Filter(filter Filter[T]) SliceIter[T] {
	return func(yield func(T) bool) {
		i(func(element T) bool {
			if filter(element) {
				if !yield(element) {
					return false
				}
			}
			return true
		})
	}
}

func (i SliceIter[T]) ForEach(handler func(val T) bool) {
	if i == nil {
		return
	}
	i(handler)
}

func (i SliceIter[T]) Count() int {
	if i == nil {
		return 0
	}
	var count int
	i(func(_ T) bool {
		count++
		return true
	})
	return count
}

func (i SliceIter[T]) WithIndex() MapIter[int, T] {
	return func(yield func(int, T) bool) {
		idx := 0
		i(func(val T) bool {
			shouldContinue := yield(idx, val)
			idx++
			return shouldContinue
		})
	}
}

func (i SliceIter[T]) Limit(limit int) SliceIter[T] {
	if limit <= 0 {
		return func(yield func(T) bool) {}
	}
	return func(yield func(T) bool) {
		count := 0
		i(func(val T) bool {
			if !yield(val) {
				return false
			}
			count++
			return count < limit
		})
	}
}

func (i SliceIter[T]) First() (T, bool) {
	var (
		val   T
		found bool
	)
	i(func(v T) bool {
		val = v
		found = true
		return false
	})
	return val, found
}

// PartitionSlice will break a [SliceIter] into groups based on the result of partitionFn.
func PartitionSlice[T any, K comparable](i SliceIter[T], partitionFn func(T) K) map[K]SliceIter[T] {
	segments := map[K][]T{}
	i(func(val T) bool {
		key := partitionFn(val)
		slice := segments[key]
		segments[key] = append(slice, val)
		return true
	})
	partitioned := map[K]SliceIter[T]{}
	for key, slice := range segments {
		partitioned[key] = SelectAll(slice)
	}
	return partitioned
}
