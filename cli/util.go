package cli

import (
	"errors"
	"fmt"
)

// MustGet is used with a [pflag.FlagSet] getter to panic if the flag is not defined, or is not the right type.
// The developer usually knows whether a get call will fail, so this function makes it easier to avoid global flag state.
func MustGet[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

var (
	ErrArgMap = errors.New("failed to map argument(s)")
)

// MapArgs is an easy way to map arguments to variables (targets), and require a certain amount.
// This will return an error if there are not enough args and/or targets to satisfy the amount required by minArgs.
// Targets elements should not be nil.
func MapArgs(args []string, minArgs int, targets ...*string) error {
	if len(args) < minArgs {
		return fmt.Errorf("%w: not enough arguments (%d) to satisfy minArgs (%d)", ErrArgMap, len(args), minArgs)
	}
	if len(targets) < minArgs {
		return fmt.Errorf("%w: not enough targets (%d) to satisfy minArgs (%d)", ErrArgMap, len(targets), minArgs)
	}
	for i := 0; i < len(args) && i < len(targets); i++ {
		if targets[i] == nil {
			return fmt.Errorf("%w: target %d is nil", ErrArgMap, i)
		}
		*targets[i] = args[i]
	}
	return nil
}
