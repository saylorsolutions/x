package bidimap

import "sync"

// BidiMap represents a generic, concurrency safe, bidirectional map between key and value.
// The keys and values are stored twice, once in each mapping.
type BidiMap[K comparable, V comparable] struct {
	mux  sync.Mutex
	ktov map[K]V
	vtok map[V]K
}

// New creates a new [BidiMap] and initializes the internal maps.
// This isn't strictly required, because non-nil instances will be initialized upon first use anyway.
func New[K comparable, V comparable]() *BidiMap[K, V] {
	m := new(BidiMap[K, V])
	m.init()
	return m
}

func (m *BidiMap[K, V]) init() {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m == nil {
		panic("nil BidiMap!")
	}
	if m.ktov == nil {
		m.ktov = map[K]V{}
	}
	if m.vtok == nil {
		m.vtok = map[V]K{}
	}
}

func (m *BidiMap[K, V]) Add(key K, val V) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	m.ktov[key] = val
	m.vtok[val] = key
}

func (m *BidiMap[K, V]) ValueOk(key K) (V, bool) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	val, ok := m.ktov[key]
	return val, ok
}

func (m *BidiMap[K, V]) Value(key K) V {
	val, _ := m.ValueOk(key)
	return val
}

func (m *BidiMap[K, V]) HasValue(val V) bool {
	_, ok := m.KeyOk(val)
	return ok
}

func (m *BidiMap[K, V]) KeyOk(value V) (K, bool) {
	m.init()
	m.mux.Lock()
	defer m.mux.Unlock()
	val, ok := m.vtok[value]
	return val, ok
}

func (m *BidiMap[K, V]) Key(value V) K {
	val, _ := m.KeyOk(value)
	return val
}

func (m *BidiMap[K, V]) HasKey(key K) bool {
	_, ok := m.ValueOk(key)
	return ok
}
