//go:build !noassert

package assert

import (
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
)

var disabled atomic.Bool

// Disable will disable assertion evaluation globally.
// This is concurrency safe, but can have side effects in other goroutines that use assertions.
func Disable() {
	disabled.Store(true)
}

// Enable can be used to re-enable assertion evaluation if Disable was called previously.
// Note that this is a global setting, and calling Disable or Enable can have unintended side effects in other goroutines that use assertions.
func Enable() {
	disabled.Store(false)
}

func getCallerDetails() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("'%s#%d'", file, line)
}

// True will panic with descriptive information if result is not true.
func True(label string, result bool) {
	if disabled.Load() {
		return
	}
	if !result {
		panic(fmt.Sprintf("assertion '%s' failed at %s", label, getCallerDetails()))
	}
}

// TrueFunc will panic with descriptive information if assertion returns false.
func TrueFunc(label string, assertion func() bool) {
	if disabled.Load() {
		return
	}
	result := assertion()
	if !result {
		panic(fmt.Sprintf("assertion '%s' failed at %s", label, getCallerDetails()))
	}
}

// NotEmpty will panic if the length of the given value is 0.
// Note that types that can't return a length using [reflect.Value.Len] will result in this function always panicking.
func NotEmpty(label string, val any) {
	if disabled.Load() {
		return
	}
	if reflect.ValueOf(val).Len() == 0 {
		panic(fmt.Sprintf("assertion '%s' failed at %s", label, getCallerDetails()))
	}
}
