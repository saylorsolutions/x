package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func getEnv() map[string]string {
	envMap := map[string]string{}
	environ := os.Environ()
	for i := 0; i < len(environ); i++ {
		key, val, found := strings.Cut(environ[i], "=")
		if !found {
			continue
		}
		envMap[strings.ToLower(key)] = val
	}
	return envMap
}

// Val will attempt to get an environment variable value using the given key.
// If the variable isn't set, or is empty, then the defaultVal will be returned.
// Note that keys are compared case-insensitive.
func Val(key string, defaultVal string) string {
	envMap := getEnv()
	key = strings.ToLower(key)

	if val, ok := envMap[key]; ok {
		trimmed := strings.TrimSpace(val)
		if len(trimmed) == 0 {
			return defaultVal
		}
		return trimmed
	}
	return defaultVal
}

// BoolIf allows translating an environment variable string value to a boolean using the given translation map.
// It's expected for the user to populate translation with a set of strings that relate to the map key.
// A whitelist for one particular value can be created by setting either the true or false slice to be empty.
// These values will be compared in a case-insensitive way.
//
// The defaultVal will be returned if the variable isn't set, is empty, or can't be a boolean value.
//
// If instead you'd like to use a custom translation policy for [Bool], try changing the slices [DefaultTrue] and [DefaultFalse].
func BoolIf(key string, defaultVal bool, translation map[bool][]string) bool {
	sval := strings.ToLower(Val(key, ""))
	if len(sval) == 0 {
		return defaultVal
	}
	if translation == nil {
		return defaultVal
	}
	truthy := translation[true]
	falsy := translation[false]
	val, ok := asBool(sval, truthy, falsy)
	if !ok {
		return defaultVal
	}
	return val
}

var (
	DefaultTrue  = []string{"1", "yes", "true", "on"}  // DefaultTrue are the values considered "true" when using [Bool], and can be changed.
	DefaultFalse = []string{"0", "no", "false", "off"} // DefaultFalse are the values considered "false" when using [Bool], and can be changed.
)

// Bool interprets an environment variable as a boolean, using [DefaultTrue] and [DefaultFalse].
// The defaultVal will be returned if the variable isn't set, is empty, or can't be a boolean value.
func Bool(key string, defaultVal bool) bool {
	sval := Val(key, "")
	val, ok := AsBool(sval)
	if !ok {
		return defaultVal
	}
	return val
}

// AsBool is an interpreter function that is used internally in [Bool], and can be passed to [InterpretSlice].
func AsBool(sval string) (bool, bool) {
	return asBool(sval, DefaultTrue, DefaultFalse)
}

func asBool(sval string, truthy []string, falsy []string) (bool, bool) {
	for i := 0; i < len(truthy); i++ {
		if sval == strings.ToLower(truthy[i]) {
			return true, true
		}
	}
	for i := 0; i < len(falsy); i++ {
		if sval == strings.ToLower(falsy[i]) {
			return false, true
		}
	}
	return false, false
}

// Int will attempt to interpret an environment variable as an integer, returning the defaultVal if the environment variable isn't found or can't be a valid integer.
func Int(key string, defaultVal int64) int64 {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	ival, ok := AsInt(sval)
	if !ok {
		return defaultVal
	}
	return ival
}

// AsInt is an interpreter function that is used internally in [Int], and can be passed to [InterpretSlice].
func AsInt(sval string) (int64, bool) {
	ival, err := strconv.ParseInt(sval, 10, 64)
	if err != nil {
		return 0, false
	}
	return ival, true
}

// Float will attempt to interpret an environment variable as a float64, returning the defaultVal if the environment variable isn't found or can't be a valid float64.
func Float(key string, defaultVal float64) float64 {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	fval, ok := AsFloat(sval)
	if !ok {
		return defaultVal
	}
	return fval
}

// AsFloat is an interpreter function that is used internally in [Float], and can be passed to [InterpretSlice].
func AsFloat(val string) (float64, bool) {
	fval, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0, false
	}
	return fval, true
}

// Duration will attempt to interpret an environment variable as a [time.Duration], returning the defaultVal if the environment variable isn't found or can't be a valid [time.Duration].
func Duration(key string, defaultVal time.Duration) time.Duration {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	dval, ok := AsDuration(sval)
	if !ok {
		return defaultVal
	}
	return dval
}

// AsDuration is an interpreter function that is used internally in [Duration], and can be passed to [InterpretSlice].
func AsDuration(val string) (time.Duration, bool) {
	dur, err := time.ParseDuration(val)
	if err != nil {
		return time.Duration(0), false
	}
	return dur, true
}

// InterpretSlice interprets an environment variable as a slice of some type T.
// If the variable is defined and non-empty, then the delimiter is used to separate out the segments of the slice value.
// The terp function is then used to interpret each segment of the split string.
// Any values that cannot be interpreted by the function will be set to the default value.
func InterpretSlice[T any](key string, delimiter string, defaultVal T, terp func(val string) (T, bool)) []T {
	sval := Val(key, "")
	if len(sval) == 0 {
		return nil
	}
	elements := strings.Split(sval, delimiter)
	slice := make([]T, len(elements))
	for i := 0; i < len(elements); i++ {
		eval, ok := terp(strings.TrimSpace(elements[i]))
		if !ok {
			slice[i] = defaultVal
			continue
		}
		slice[i] = eval
	}
	return slice
}

// ValSlice will interpret the environment variable named key as a slice of strings.
// Any empty values will be set to the default value.
func ValSlice(key string, delimiter string, defaultVal string) []string {
	return InterpretSlice(key, delimiter, defaultVal, func(val string) (string, bool) {
		if len(val) == 0 {
			return "", false
		}
		return val, true
	})
}

// BoolSlice will interpret the environment variable named key as a slice of booleans.
// Any non-boolean values will be set to the default value.
func BoolSlice(key string, delimiter string, defaultVal bool) []bool {
	return InterpretSlice(key, delimiter, defaultVal, AsBool)
}

// IntSlice will interpret the environment variable named key as a slice of int64s.
// Any non-int values will be set to the default value.
func IntSlice(key string, delimiter string, defaultVal int64) []int64 {
	return InterpretSlice(key, delimiter, defaultVal, AsInt)
}

// FloatSlice will interpret the environment variable named key as a slice of float64s.
// Any non-float values will be set to the default value.
func FloatSlice(key string, delimiter string, defaultVal float64) []float64 {
	return InterpretSlice(key, delimiter, defaultVal, AsFloat)
}

// DurationSlice will interpret the environment variable named key as a slice of durations.
// Any non-duration values will be set to the default value.
func DurationSlice(key string, delimiter string, defaultVal time.Duration) []time.Duration {
	return InterpretSlice(key, delimiter, defaultVal, AsDuration)
}
