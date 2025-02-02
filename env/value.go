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
	if trueVals, ok := translation[true]; ok && len(trueVals) > 0 {
		for i := 0; i < len(trueVals); i++ {
			if sval == strings.ToLower(trueVals[i]) {
				return true
			}
		}
	}
	if falseVals, ok := translation[false]; ok && len(falseVals) > 0 {
		for i := 0; i < len(falseVals); i++ {
			if sval == strings.ToLower(falseVals[i]) {
				return false
			}
		}
	}
	return defaultVal
}

var (
	DefaultTrue  = []string{"1", "yes", "true", "on"}  // DefaultTrue are the values considered "true" when using [Bool], and can be changed.
	DefaultFalse = []string{"0", "no", "false", "off"} // DefaultFalse are the values considered "false" when using [Bool], and can be changed.
)

// Bool interprets an environment variable as a boolean, using [DefaultTrue] and [DefaultFalse].
// The defaultVal will be returned if the variable isn't set, is empty, or can't be a boolean value.
func Bool(key string, defaultVal bool) bool {
	return BoolIf(key, defaultVal, map[bool][]string{
		true:  DefaultTrue,
		false: DefaultFalse,
	})
}

// Int will attempt to interpret an environment variable as an integer, returning the defaultVal if the environment variable isn't found or can't be a valid integer.
func Int(key string, defaultVal int64) int64 {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	ival, err := strconv.ParseInt(sval, 10, 64)
	if err != nil {
		return defaultVal
	}
	return ival
}

// Float will attempt to interpret an environment variable as a float64, returning the defaultVal if the environment variable isn't found or can't be a valid float64.
func Float(key string, defaultVal float64) float64 {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	fval, err := strconv.ParseFloat(sval, 64)
	if err != nil {
		return defaultVal
	}
	return fval
}

// Duration will attempt to interpret an environment variable as a [time.Duration], returning the defaultVal if the environment variable isn't found or can't be a valid [time.Duration].
func Duration(key string, defaultVal time.Duration) time.Duration {
	sval := Val(key, "")
	if len(sval) == 0 {
		return defaultVal
	}
	dval, err := time.ParseDuration(sval)
	if err != nil {
		return defaultVal
	}
	return dval
}
