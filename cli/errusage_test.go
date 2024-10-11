package cli

import (
	"errors"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestUsageError_Is(t *testing.T) {
	err := NewUsageError("test")
	assert.ErrorIs(t, err, &UsageError{})

	var ErrTesting = errors.New("test")
	err2 := NewUsageError("%w", ErrTesting)
	assert.ErrorIs(t, err2, &UsageError{})
	assert.ErrorIs(t, err2, ErrTesting)
}

func TestUsageError_Unwrap(t *testing.T) {
	var ErrTesting = errors.New("test")
	err := NewUsageError("%w", ErrTesting)
	var targetUsage = new(UsageError)
	assert.True(t, errors.As(err, &targetUsage))
}

func TestUsageError_Error(t *testing.T) {
	err := &UsageError{}
	assert.Equal(t, "usage error", err.Error(), "Default error output should be returned when there is no wrapping error")
	err2 := NewUsageError("test")
	assert.Equal(t, "usage error: test", err2.Error(), "The wrapped error's output should be returned when Error is called")
}

func ExampleNewUsageError() {
	tlc := NewCommandSet("parent")
	cmd := tlc.AddCommand("command", "test command")
	cmd.Does(func(flags *flag.FlagSet, out *Printer) error {
		return NewUsageError("test usage error")
	})
	// Done for testing purposes
	cmd.Printer().Redirect(os.Stdout)
	// Error not handled for brevity
	_ = tlc.Exec([]string{"command"})

	// Output:
	// usage error: test usage error
	//
	// test command
	//
	// USAGE:
	// parent command
	//
	// FLAGS
	//   -h, --help   Prints this usage information
}
