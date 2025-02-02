// Package iterx provides some extensions to the base [iter.Seq] and [iter.Seq2] types.
package iterx

import (
	"iter"
)

// Filter is a function that returns true if the element of an [iter.Seq] should be yielded to the caller.
type Filter[T any] func(T) bool

// NoZeroValues creates a [Filter] that excludes elements that are the zero value for the type.
// The underlying type must be comparable.
func NoZeroValues[T comparable]() Filter[T] {
	var mt T
	return func(val T) bool {
		return val != mt
	}
}

// NotEqual creates a [Filter] that excludes elements that equal the value.
// The underlying type must be comparable.
func NotEqual[T comparable](val T) Filter[T] {
	return func(el T) bool {
		return el != val
	}
}

// And combines multiple [Filter] into one, where both must be true to yield the element.
func (f Filter[T]) And(other Filter[T]) Filter[T] {
	return func(element T) bool {
		if f(element) && other(element) {
			return true
		}
		return false
	}
}

// Or combines multiple [Filter] into one, where one or the other must be true to yield the element.
func (f Filter[T]) Or(other Filter[T]) Filter[T] {
	return func(element T) bool {
		if f(element) || other(element) {
			return true
		}
		return false
	}
}

// Any will create a [Filter] that matches all elements.
func Any[T any]() Filter[T] {
	return func(element T) bool {
		return true
	}
}

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

type MapIter[K comparable, V any] iter.Seq2[K, V]

// SelectMap translates any map to a [MapIter].
func SelectMap[K comparable, V any](m map[K]V) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		for key, value := range m {
			if !yield(key, value) {
				break
			}
		}
	}
}

// SelectSet will translate a map as a set of values into a [MapIter].
// This is semantically equivalent to calling SelectMap(m).Keys(), but slightly more computationally efficient.
func SelectSet[K comparable, V any](m map[K]V) SliceIter[K] {
	return func(yield func(K) bool) {
		for key := range m {
			if !yield(key) {
				break
			}
		}
	}
}

// SliceMap will create a [MapIter] with the indexes and values of the given slice.
func SliceMap[T any](slice []T) MapIter[int, T] {
	return func(yield func(int, T) bool) {
		for i, val := range slice {
			if !yield(i, val) {
				return
			}
		}
	}
}

func (i MapIter[K, V]) Map() map[K]V {
	m := map[K]V{}
	i(func(key K, val V) bool {
		m[key] = val
		return true
	})
	return m
}

func (i MapIter[K, V]) FilterKeys(filter Filter[K]) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		if filter == nil {
			panic("nil filter")
		}
		i(func(key K, value V) bool {
			if filter(key) {
				return yield(key, value)
			}
			return true
		})
	}
}

func (i MapIter[K, V]) FilterValues(filter Filter[V]) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		if filter == nil {
			panic("nil filter")
		}
		i(func(key K, value V) bool {
			if filter(value) {
				return yield(key, value)
			}
			return true
		})
	}
}

func (i MapIter[K, V]) Keys() SliceIter[K] {
	return func(yield func(K) bool) {
		i(func(key K, _ V) bool {
			return yield(key)
		})
	}
}

func (i MapIter[K, V]) Values() SliceIter[V] {
	return func(yield func(V) bool) {
		i(func(_ K, value V) bool {
			return yield(value)
		})
	}
}

func (i MapIter[K, V]) ForEach(handler func(key K, val V) bool) {
	if i == nil {
		return
	}
	i(handler)
}

func (i MapIter[K, V]) Count() int {
	if i == nil {
		return 0
	}
	var count int
	i.ForEach(func(_ K, _ V) bool {
		count++
		return true
	})
	return count
}

func (i MapIter[K, V]) Limit(limit int) MapIter[K, V] {
	if limit <= 0 {
		return func(yield func(K, V) bool) {}
	}
	return func(yield func(K, V) bool) {
		count := 0
		i(func(key K, val V) bool {
			if !yield(key, val) {
				return false
			}
			count++
			return count < limit
		})
	}
}

func (i MapIter[K, V]) First() (K, V, bool) {
	var (
		key   K
		val   V
		found bool
	)
	i(func(k K, v V) bool {
		key = k
		val = v
		found = true
		return false
	})
	return key, val, found
}
