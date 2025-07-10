package iterx

import "iter"

type SliceIter[T any] iter.Seq[T]

// SelectAll will translate any slice into a [SliceIter].
// Deprecated: use Select with no filters instead.
func SelectAll[T any](slice []T) SliceIter[T] {
	return Select(slice)
}

func SelectValue[T any](value T) SliceIter[T] {
	return func(yield func(T) bool) {
		yield(value)
	}
}

// Select will create a SliceIter using any provided [Filter] to select elements from the slice.
func Select[T any](slice []T, filters ...Filter[T]) SliceIter[T] {
	if len(slice) == 0 {
		return func(yield func(T) bool) {}
	}
	filter := Any[T]()
	for _, given := range filters {
		filter = filter.And(given)
	}
	return func(yield func(T) bool) {
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

// Append will concatenate two SliceIter to represent them as one.
//
// Note that this does not mutate the original SliceIter, but produces a new one.
func (i SliceIter[T]) Append(next SliceIter[T]) SliceIter[T] {
	return func(yield func(T) bool) {
		sendNext := true
		i(func(val T) bool {
			sendNext = yield(val)
			return sendNext
		})
		if sendNext {
			next(func(val T) bool {
				return yield(val)
			})
		}
	}
}

// AppendValue will concatenate the given value onto the end of this SliceIter.
//
// Note that this does not mutate the original SliceIter, but produces a new one.
func (i SliceIter[T]) AppendValue(value T) SliceIter[T] {
	return i.Append(func(yield func(T) bool) {
		yield(value)
	})
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

// WithIndex returns a MapIter that attaches a zero-based index to each value.
// This doesn't necessarily match the original slice indexes if the SliceIter is filtered.
//
// Since this is based on slice iteration, this MapIter will have a predictable iteration order.
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

func (i SliceIter[T]) Offset(offset int) SliceIter[T] {
	if offset <= 0 {
		return i
	}
	return func(yield func(T) bool) {
		var skipped int
		i(func(val T) bool {
			if skipped < offset {
				skipped++
				return true
			}
			return yield(val)
		})
	}
}

func (i SliceIter[T]) Limit(limit int) SliceIter[T] {
	if limit <= 0 {
		return func(yield func(T) bool) {}
	}
	return func(yield func(T) bool) {
		var yielded int
		i(func(val T) bool {
			if yielded >= limit {
				return false
			}
			yielded++
			return yield(val)
		})
	}
}

func (i SliceIter[T]) First() (first T, found bool) {
	i(func(val T) bool {
		first, found = val, true
		return false
	})
	return
}

func (i SliceIter[T]) Last() (last T, found bool) {
	i(func(val T) bool {
		last, found = val, true
		return true
	})
	return
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
		partitioned[key] = Select(slice)
	}
	return partitioned
}

// TransformSlice will transform each selected value in a [SliceIter] into a value in a new [SliceIter].
func TransformSlice[A any, B any](iter SliceIter[A], transform func(in A) B) SliceIter[B] {
	return func(yield func(B) bool) {
		iter(func(input A) bool {
			return yield(transform(input))
		})
	}
}

// TransformSliceToMap will transform any slice into a MapIter using the transform function.
// Resulting map keys will be deduplicated.
func TransformSliceToMap[T any, K comparable, V any](slice []T, transform func(val T) (K, V)) MapIter[K, V] {
	if len(slice) == 0 {
		return func(yield func(K, V) bool) {}
	}
	return DedupeKeys(func(yield func(K, V) bool) {
		for _, val := range slice {
			if !yield(transform(val)) {
				return
			}
		}
	})
}

func DedupeSlice[T comparable](slice SliceIter[T]) SliceIter[T] {
	return func(yield func(T) bool) {
		seen := map[T]bool{}
		slice(func(val T) bool {
			if seen[val] {
				return true
			}
			seen[val] = true
			return yield(val)
		})
	}
}
