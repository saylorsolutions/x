package bidimap

import (
	"github.com/saylorsolutions/x/structures/set"
	"sync"
)

// MultiMap is a generic, many-to-many, bidirectional, concurrency safe map between key and value.
// The keys and values are stored twice for each association between one key and one value, once for each related key and set.
// With many associations, this structure can result in a quadratic increase in memory usage, but lookups will remain O(1).
type MultiMap[K comparable, V comparable] struct {
	mux  sync.Mutex
	ktov map[K]set.Set[V]
	vtok map[V]set.Set[K]
}

// NewMulti creates a new [MultiMap] and initializes the internal maps.
// This isn't strictly required, because non-nil instances will be initialized upon first use anyway.
func NewMulti[K comparable, V comparable]() *MultiMap[K, V] {
	mm := new(MultiMap[K, V])
	mm.init()
	return mm
}

func (m *MultiMap[K, V]) init() {
	if m == nil {
		panic("nil MultiMap")
	}
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.ktov == nil {
		m.ktov = map[K]set.Set[V]{}
	}
	if m.vtok == nil {
		m.vtok = map[V]set.Set[K]{}
	}
}

func (m *MultiMap[K, V]) AddValues(key K, value V, others ...V) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	m.ktov[key] = m.ktov[key].Add(value)
	m.vtok[value] = m.vtok[value].Add(key)
	for _, other := range others {
		m.ktov[key] = m.ktov[key].Add(other)
		m.vtok[other] = m.vtok[other].Add(key)
	}
}

func (m *MultiMap[K, V]) AddKeys(value V, key K, others ...K) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	m.vtok[value] = m.vtok[value].Add(key)
	m.ktov[key] = m.ktov[key].Add(value)
	for _, other := range others {
		m.vtok[value] = m.vtok[value].Add(other)
		m.ktov[other] = m.ktov[other].Add(value)
	}
}

// GetValuesOk will return a slice of values associated with the key, if it exists.
func (m *MultiMap[K, V]) GetValuesOk(key K) ([]V, bool) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	_set := m.ktov[key]
	return _set.Slice(), _set != nil && len(_set) > 0
}

// GetValues will return a slice of values associated with the given key.
// The values will not be in a consistent order.
func (m *MultiMap[K, V]) GetValues(key K) []V {
	vs, _ := m.GetValuesOk(key)
	return vs
}

// GetValueSet will return a copy of the underlying value set associated with the given key.
func (m *MultiMap[K, V]) GetValueSet(key K) set.Set[V] {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.ktov[key].Copy()
}

func (m *MultiMap[K, V]) HasValue(value V) bool {
	_, ok := m.GetKeysOk(value)
	return ok
}

// GetKeysOk will return a slice of keys associated with the value, if it exists.
func (m *MultiMap[K, V]) GetKeysOk(value V) ([]K, bool) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	_set := m.vtok[value]
	return _set.Slice(), _set != nil && len(_set) > 0
}

// GetKeys will return a slice of keys associated with the given value.
// The keys will not be in a consistent order.
func (m *MultiMap[K, V]) GetKeys(value V) []K {
	ks, _ := m.GetKeysOk(value)
	return ks
}

// GetKeySet will return a copy of the underlying key set associated with the given value.
func (m *MultiMap[K, V]) GetKeySet(value V) set.Set[K] {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.vtok[value].Copy()
}

func (m *MultiMap[K, V]) HasKey(key K) bool {
	_, ok := m.GetValuesOk(key)
	return ok
}
