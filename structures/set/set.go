package set

// Set formalizes set semantics for a
type Set[T comparable] map[T]struct{}

// New creates a new [Set] from the given values.
// The returned [Set] will have no values if none are given.
func New[T comparable](vals ...T) Set[T] {
	s := Set[T]{}
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// FromKeys will create a new [Set] from the keys of the given map, if any are present.
func FromKeys[T comparable, E any](vals map[T]E) Set[T] {
	s := Set[T]{}
	if vals == nil {
		return s
	}
	for v := range vals {
		s[v] = struct{}{}
	}
	return s
}

func (s Set[T]) Slice() []T {
	if len(s) == 0 {
		return nil
	}
	vals := make([]T, len(s))
	i := 0
	for val := range s {
		vals[i] = val
		i++
	}
	return vals
}

func (s Set[T]) Add(val T, others ...T) Set[T] {
	if s == nil {
		s = Set[T]{}
	}
	s[val] = struct{}{}
	for _, v := range others {
		s[v] = struct{}{}
	}
	return s
}

func (s Set[T]) Remove(val T, others ...T) Set[T] {
	if s == nil {
		s = Set[T]{}
	}
	delete(s, val)
	for _, v := range others {
		delete(s, v)
	}
	return s
}

func (s Set[T]) Has(val T) bool {
	_, ok := s[val]
	return ok
}

// HasAny determines if any of the given values are present in the [Set].
// If the parameter list is empty, then false is returned.
func (s Set[T]) HasAny(values ...T) bool {
	if len(s) == 0 {
		return false
	}
	for _, value := range values {
		if s.Has(value) {
			return true
		}
	}
	return false
}

// HasAll determines if all given values are present in the [Set].
// If the parameter list is empty, then false is returned.
func (s Set[T]) HasAll(values ...T) bool {
	if len(s) == 0 {
		return false
	}
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !s.Has(value) {
			return false
		}
	}
	return true
}

// Intersection returns a new [Set] with only the values common between sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	inter := Set[T]{}
	for v := range other {
		if s.Has(v) {
			inter.Add(v)
		}
	}
	return inter
}

// Difference returns a new [Set] with the common values between sets removed.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	diff := Set[T]{}
	for v := range s {
		if !other.Has(v) {
			diff.Add(v)
		}
	}
	return diff
}

// Union returns a new [Set] with all values from both sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	union := Set[T]{}
	for v := range s {
		union.Add(v)
	}
	for v := range other {
		union.Add(v)
	}
	return union
}

func (s Set[T]) Copy() Set[T] {
	return New[T](s.Slice()...)
}
