package assert

import (
	"fmt"
	"strings"
)

// Collector collects errors and can join them with the specified join string.
// This is a little bit more convenient than maintaining a slice and using [errors.Join].
//
// A Collector is itself an error, so it can be returned directly and compared with [errors.Is] or [errors.As].
//
// Note that a Collector is not concurrency safe.
type Collector struct {
	errs    []error
	joinStr string
}

// CollectErrors creates a new Collector, optionally with a join string that differs from the default of "\n".
func CollectErrors(joinString ...string) *Collector {
	joinStr := "\n"
	if len(joinString) > 0 {
		joinStr = joinString[0]
	}
	return &Collector{
		joinStr: joinStr,
	}
}

// Add adds a new, potentially nil error to the Collector.
// Nil errors will not be included.
func (c *Collector) Add(err error) *Collector {
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return c
}

// AddString allows creating an error string using [fmt.Errorf], which means that the "%w" format string may be used.
func (c *Collector) AddString(msg string, args ...any) *Collector {
	return c.Add(fmt.Errorf(msg, args...))
}

// Result will return nil if no errors have been added to the Collector.
// Otherwise, it will return itself.
//
// This is provided because returning an empty Collector is still returning a non-nil error.
func (c *Collector) Result() error {
	if len(c.errs) > 0 {
		return c
	}
	return nil
}

// Error satisfies the error interface.
func (c *Collector) Error() string {
	var buf strings.Builder
	for i, err := range c.errs {
		if i > 0 {
			buf.WriteString(c.joinStr)
		}
		buf.WriteString(err.Error())
	}
	return buf.String()
}

// Unwrap allows using [errors.Is] and [errors.As] to identify any error in the Collector.
func (c *Collector) Unwrap() []error {
	return c.errs
}
