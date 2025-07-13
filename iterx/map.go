// Package iterx provides some extensions to the base [iter.Seq] and [iter.Seq2] types.
package iterx

import (
	"iter"
)

type MapIter[K comparable, V any] iter.Seq2[K, V]

// SelectMap translates any map to a [MapIter].
func SelectMap[K comparable, V any](m map[K]V) MapIter[K, V] {
	if len(m) == 0 {
		return func(yield func(K, V) bool) {}
	}
	return func(yield func(K, V) bool) {
		for key, value := range m {
			if !yield(key, value) {
				break
			}
		}
	}
}

func SelectEntry[K comparable, V any](key K, value V) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		yield(key, value)
	}
}

// SliceMap will create a [MapIter] with the indexes and values of the given slice.
// This MapIter is naturally ordered.
func SliceMap[T any](slice []T) MapIter[int, T] {
	return func(yield func(int, T) bool) {
		for i, val := range slice {
			if !yield(i, val) {
				return
			}
		}
	}
}

// SliceInverseMap will create a MapIter where slice values are keys, and their indexes are values.
//
// If there are duplicate elements in the slice, then the index of the first duplicate value will be retained.
// This is to ensure that keys are still unique to satisfy normal map semantics.
func SliceInverseMap[T comparable](slice []T) MapIter[T, int] {
	return TransformEntries(Select(slice).WithIndex(), func(key int, val T) (T, int) {
		return val, key
	})
}

// SliceSet will create a [MapIter] with all distinct values in the given slice.
func SliceSet[T comparable](slice []T) MapIter[T, bool] {
	m := map[T]bool{}
	for _, val := range slice {
		m[val] = true
	}
	return SelectMap(m)
}

func (i MapIter[K, V]) Map() map[K]V {
	m := map[K]V{}
	i(func(key K, val V) bool {
		m[key] = val
		return true
	})
	return m
}

func (i MapIter[K, V]) Filter(entryFilter func(key K, val V) bool) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		i(func(key K, val V) bool {
			if entryFilter(key, val) {
				yield(key, val)
			}
			return true
		})
	}
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

// Append will concatenate two MapIter to represent them as one.
// Duplicate keys will be removed.
//
// Note that this does not mutate the original MapIter, but produces a new one.
func (i MapIter[K, V]) Append(next MapIter[K, V]) MapIter[K, V] {
	return DedupeKeys(func(yield func(K, V) bool) {
		sendNext := true
		i(func(key K, val V) bool {
			sendNext = yield(key, val)
			return sendNext
		})
		if sendNext {
			next(func(key K, val V) bool {
				return yield(key, val)
			})
		}
	})
}

// AppendEntry will concatenate the given key/value pair onto the end of this MapIter.
// Duplicate keys will be removed.
//
// Note that this does not mutate the original MapIter, but produces a new one.
func (i MapIter[K, V]) AppendEntry(key K, value V) MapIter[K, V] {
	return i.Append(func(yield func(K, V) bool) {
		yield(key, value)
	})
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

func (i MapIter[K, V]) KeyOrder(order func(keys SliceIter[K]) SliceIter[K]) MapIter[K, V] {
	m := i.Map()
	keys := order(i.Keys())
	return func(yield func(K, V) bool) {
		keys(func(key K) bool {
			return yield(key, m[key])
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

// Offset will skip the first N records during iteration.
//
// Note that iteration order is not guaranteed without a previous call to KeyOrder.
func (i MapIter[K, V]) Offset(offset int) MapIter[K, V] {
	if offset <= 0 {
		return i
	}
	return func(yield func(K, V) bool) {
		var skipped int
		i(func(key K, val V) bool {
			if skipped < offset {
				skipped++
				return true
			}
			return yield(key, val)
		})
	}
}

// Limit will limit how many records are returned.
//
// Note that iteration order is not guaranteed without a previous call to KeyOrder.
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

func (i MapIter[K, V]) First() (firstKey K, firstVal V, found bool) {
	i(func(key K, val V) bool {
		firstKey, firstVal, found = key, val, true
		return false
	})
	return
}

func (i MapIter[K, V]) Last() (lastKey K, lastVal V, found bool) {
	i(func(key K, val V) bool {
		lastKey, lastVal, found = key, val, true
		return true
	})
	return
}

func (i MapIter[K, V]) HasKey(key K) (result bool) {
	_, _, ok := i.FilterKeys(Equal(key)).First()
	return ok
}

func TransformEntries[K1 comparable, K2 comparable, V1 any, V2 any](input MapIter[K1, V1], transform func(key K1, val V1) (K2, V2)) MapIter[K2, V2] {
	return DedupeKeys[K2, V2](
		func(yield func(K2, V2) bool) {
			input.ForEach(func(key K1, val V1) bool {
				return yield(transform(key, val))
			})
		},
	)
}

func TransformKeys[K1 comparable, K2 comparable, V any](iter MapIter[K1, V], transform func(key K1) K2) MapIter[K2, V] {
	return DedupeKeys(func(yield func(K2, V) bool) {
		iter(func(key K1, val V) bool {
			return yield(transform(key), val)
		})
	})
}

func TransformValues[K comparable, V1 any, V2 any](iter MapIter[K, V1], transform func(value V1) V2) MapIter[K, V2] {
	return func(yield func(K, V2) bool) {
		iter(func(k K, v V1) bool {
			return yield(k, transform(v))
		})
	}
}

// InvertMap will produce a new [MapIter] with keys and values swapped from the original [MapIter].
//
// Since values do not need to be unique in a map, the produced MapIter's values are slices and can result in a smaller set of key/value pairs.
// This is to prevent data loss when the iterator state is collapsed into a new map.
// However, the order of original keys in the new values is not deterministic without a call to [MapIter.KeyOrder].
func InvertMap[K comparable, V comparable](iter MapIter[K, V]) MapIter[V, []K] {
	valMap := map[V]SliceIter[K]{}
	iter(func(key K, val V) bool {
		curIter := valMap[val]
		if curIter == nil {
			valMap[val] = SelectValue(key)
		} else {
			valMap[val] = curIter.AppendValue(key)
		}
		return true
	})
	return TransformValues(SelectMap(valMap), func(iter SliceIter[K]) []K {
		return iter.Slice()
	})
}

func DedupeKeys[K comparable, V any](mapIter MapIter[K, V]) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		seen := map[K]bool{}
		mapIter(func(key K, val V) bool {
			if seen[key] {
				return true
			}
			seen[key] = true
			return yield(key, val)
		})
	}
}

func DedupeValues[K comparable, V comparable](mapIter MapIter[K, V]) MapIter[K, V] {
	return func(yield func(K, V) bool) {
		seen := map[V]bool{}
		mapIter(func(key K, val V) bool {
			if seen[val] {
				return true
			}
			seen[val] = true
			return yield(key, val)
		})
	}
}
