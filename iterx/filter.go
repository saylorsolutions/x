package iterx

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

// None will create a [Filter] that doesn't match any element.
func None[T any]() Filter[T] {
	return func(_ T) bool {
		return false
	}
}
