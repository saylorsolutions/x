// Package iterx provides some extensions to the base [iter.Seq] and [iter.Seq2] types.
package iterx

import (
	"iter"
)

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
