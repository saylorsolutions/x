package eventbus

import (
	"errors"
	"fmt"
)

var (
	ErrUnexpectedTypeParam = errors.New("unexpected parameter type")
	ErrNotEnoughParams     = errors.New("not enough parameters")
)

// AssertParam is the most basic way to assert [Param] type, and is most useful when there are only 1 or 2 parameters.
// [ParamSpec] with multiple [ParamAssertion] is likely a more convenient way to make similar assertions with more parameters.
func AssertParam[T any](param Param) (T, bool) {
	if param == nil {
		var mt T
		return mt, false
	}
	val, ok := param.(T)
	if !ok {
		return val, false
	}
	return val, true
}

// ParamAssertion is a function that asserts constraints of a [Param].
// The pos parameter is informational and usually should not be the subject of an assertion.
type ParamAssertion func(pos int, p Param) error

// And is used to chain assertions into one [ParamAssertion].
// If an assertion returns an error, then execution will stop and the error will be returned.
func (a ParamAssertion) And(other ParamAssertion, more ...ParamAssertion) ParamAssertion {
	return func(pos int, p Param) error {
		if err := a(pos, p); err != nil {
			return err
		}
		if err := other(pos, p); err != nil {
			return err
		}
		for _, next := range more {
			if err := next(pos, p); err != nil {
				return err
			}
		}
		return nil
	}
}

// AnyPass will run a set of [ParamAssertion], and if any return a nil error, then execution will stop and return nil.
// This only returns an error if all [ParamAssertion] fail, and all errors will be returned.
// This is most useful if a [Param] can have one of multiple types.
func AnyPass(assertions ...ParamAssertion) ParamAssertion {
	return func(pos int, p Param) error {
		var errs []error
		for _, assertion := range assertions {
			if err := assertion(pos, p); err != nil {
				errs = append(errs, err)
				continue
			}
			// This assertion passes, so we return nil.
			return nil
		}
		return errors.Join(errs...)
	}
}

// IsType asserts that a [Param] is of the expected type.
func IsType[T Param]() ParamAssertion {
	return func(pos int, p Param) error {
		if _, ok := p.(T); !ok {
			var expected T
			return fmt.Errorf("%w: expected %T, but got %T", ErrUnexpectedTypeParam, expected, p)
		}
		return nil
	}
}

func notNil() ParamAssertion {
	return func(pos int, p Param) error {
		if p == nil {
			return fmt.Errorf("%w: parameter %d is nil", ErrUnexpectedTypeParam, pos)
		}
		return nil
	}
}

// AssertAndStore will return a [ParamAssertion] that will first assert that the [Param] is of the expected type, and then store its value in the target pointer.
// The target parameter cannot be a nil pointer.
func AssertAndStore[T any](target *T) ParamAssertion {
	if target == nil {
		return func(pos int, _ Param) error {
			return fmt.Errorf("target for param %d is nil pointer", pos)
		}
	}
	return notNil().And(IsType[T](), func(_ int, p Param) error {
		*target = p.(T)
		return nil
	})
}

// Optional can be used if a [Param] at this position is not required in all cases.
// If a non-nil [Param] is given, then the ifNotNil [ParamAssertion] will be applied.
// Optional will return a nil error otherwise.
func Optional(ifNotNil ParamAssertion) ParamAssertion {
	return func(pos int, p Param) error {
		if p == nil {
			return nil
		}
		return ifNotNil(pos, p)
	}
}

// ParamSpec uses all given [ParamAssertion] to create a function that can make assertions about all params.
// The assertion at position 0 will be applied to the [Param] at position 0, and so on for all parameters.
// If a [ParamAssertion] at a position is nil, then that [Param] will have no assertions applied to it.
// If the length of the list of assertions doesn't match the length of the list of parameters, no error is returned.
// If the number of parameters is less than minParams, then an error will be immediately returned without running any [ParamAssertion].
// This is a no-op if no assertions are passed.
func ParamSpec(minParams int, assertions ...ParamAssertion) func(params []Param) []error {
	return func(params []Param) []error {
		var errs []error
		if len(params) < minParams {
			return []error{fmt.Errorf("%w: expected at least %d parameters", ErrNotEnoughParams, minParams)}
		}
		for i := 0; i < len(assertions) && i < len(params); i++ {
			assertion := assertions[i]
			if assertion == nil {
				continue
			}
			if err := assertion(i, params[i]); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) == 0 {
			return nil
		}
		return errs
	}
}

// MapParam provides a simple interface for mapping a single parameter to a target variable, which is common for handlers.
func MapParam[T any](target *T, params []Param) error {
	errs := ParamSpec(1,
		AssertAndStore(target),
	)(params)
	return errors.Join(errs...)
}
