package env

import (
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

func tempSet(t *testing.T, key, val string) func() {
	curVal, isSet := os.LookupEnv(key)
	assert.NoError(t, os.Setenv(key, val))
	return func() {
		if isSet {
			assert.NoError(t, os.Setenv(key, curVal))
			return
		}
		assert.NoError(t, os.Unsetenv(key))
	}
}

func TestVal(t *testing.T) {
	const key = "TEST_VAL"

	tests := []struct {
		name     string
		value    string
		expected string
		unset    bool
	}{
		{
			name:     "Unset",
			unset:    true,
			expected: "default",
		},
		{
			name:     "Empty",
			value:    "",
			expected: "default",
		},
		{
			name:     "Trimmed",
			value:    "\n\t abc \t\n",
			expected: "abc",
		},
	}

	for _, tc := range tests {
		name := tc.name
		t.Run(name, func(t *testing.T) {
			if !tc.unset {
				defer tempSet(t, key, tc.value)()
			}
			result := Val(key, "default")
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBoolIf_EmptyTranslation(t *testing.T) {
	const (
		key        = "TEST_BOOLIF_EMPTY"
		defaultVal = true
	)
	defer tempSet(t, key, "true")()
	assert.NotPanics(t, func() {
		got := BoolIf(key, defaultVal, nil)
		assert.Equal(t, defaultVal, got)
	})
}

func TestBool(t *testing.T) {
	const key = "TEST_BOOL"
	tests := []struct {
		name     string
		unset    bool
		value    string
		expected bool
	}{
		{
			name:     "Unset",
			unset:    true,
			expected: false,
		},
		{
			name:     "Empty",
			value:    "",
			expected: false,
		},
		{
			name:     "Not a bool",
			value:    "blah",
			expected: false,
		},
		{
			name:     "Truthy",
			value:    DefaultTrue[0],
			expected: true,
		},
		{
			name:     "Truthy Uppercase",
			value:    strings.ToUpper(DefaultTrue[0]),
			expected: true,
		},
		{
			name:     "Falsy",
			value:    DefaultFalse[0],
			expected: false,
		},
		{
			name:     "Falsy Uppercase",
			value:    strings.ToUpper(DefaultFalse[0]),
			expected: false,
		},
	}

	for _, tc := range tests {
		name := tc.name
		t.Run(name, func(t *testing.T) {
			if !tc.unset {
				defer tempSet(t, key, tc.value)()
			}
			assert.Equal(t, tc.expected, Bool(key, false))
		})
	}
}

func TestInt(t *testing.T) {
	const (
		key              = "TEST_INT"
		defaultVal int64 = -17
	)
	tests := []struct {
		name     string
		unset    bool
		value    string
		expected int64
	}{
		{
			name:     "Unset",
			unset:    true,
			expected: defaultVal,
		},
		{
			name:     "Empty",
			value:    "",
			expected: defaultVal,
		},
		{
			name:     "Not an int",
			value:    "blah",
			expected: defaultVal,
		},
		{
			name:     "Positive",
			value:    "100",
			expected: 100,
		},
		{
			name:     "Negative",
			value:    "-100",
			expected: -100,
		},
		{
			name:     "Zero",
			value:    "0",
			expected: 0,
		},
	}

	for _, tc := range tests {
		name := tc.name
		t.Run(name, func(t *testing.T) {
			if !tc.unset {
				defer tempSet(t, key, tc.value)()
			}
			assert.Equal(t, tc.expected, Int(key, defaultVal))
		})
	}
}

func TestFloat(t *testing.T) {
	const (
		key                = "TEST_FLOAT"
		defaultVal float64 = -17
	)
	tests := []struct {
		name     string
		unset    bool
		value    string
		expected float64
	}{
		{
			name:     "Unset",
			unset:    true,
			expected: defaultVal,
		},
		{
			name:     "Empty",
			value:    "",
			expected: defaultVal,
		},
		{
			name:     "Not a float",
			value:    "blah",
			expected: defaultVal,
		},
		{
			name:     "Positive Int",
			value:    "100",
			expected: 100,
		},
		{
			// This WILL result in rounding, because 1/3 can't be accurately represented with a float.
			name:     "Rounded Fraction",
			value:    "0.333333333333333333333333",
			expected: 1.0 / 3.0,
		},
		{
			name:     "Negative Int",
			value:    "-100",
			expected: -100,
		},
		{
			name:     "Zero",
			value:    "0",
			expected: 0.0,
		},
	}

	for _, tc := range tests {
		name := tc.name
		t.Run(name, func(t *testing.T) {
			if !tc.unset {
				defer tempSet(t, key, tc.value)()
			}
			assert.Equal(t, tc.expected, Float(key, defaultVal))
		})
	}
}

func TestDuration(t *testing.T) {
	const (
		key                      = "TEST_DUR"
		defaultVal time.Duration = -5 * time.Minute
	)
	tests := []struct {
		name     string
		unset    bool
		value    string
		expected time.Duration
	}{
		{
			name:     "Unset",
			unset:    true,
			expected: defaultVal,
		},
		{
			name:     "Empty",
			value:    "",
			expected: defaultVal,
		},
		{
			name:     "Not a duration",
			value:    "blah",
			expected: defaultVal,
		},
		{
			name:     "Positive",
			value:    "10m",
			expected: 10 * time.Minute,
		},
		{
			name:     "Negative",
			value:    "-10m",
			expected: -10 * time.Minute,
		},
		{
			name:     "Zero",
			value:    "0h",
			expected: 0,
		},
	}

	for _, tc := range tests {
		name := tc.name
		t.Run(name, func(t *testing.T) {
			if !tc.unset {
				defer tempSet(t, key, tc.value)()
			}
			assert.Equal(t, tc.expected, Duration(key, defaultVal))
		})
	}
}

func TestInterpretSlice(t *testing.T) {
	const key = "SOME_ENV_VAL"

	t.Run("String slice", func(t *testing.T) {
		defer tempSet(t, key, "a:b:c:")()
		vals := ValSlice(key, ":", "d")
		assert.Equal(t, []string{"a", "b", "c", "d"}, vals)
	})
	t.Run("Bool slice", func(t *testing.T) {
		defer tempSet(t, key, "true,blue,1,off,")()
		vals := BoolSlice(key, ",", false)
		assert.Equal(t, []bool{true, false, true, false, false}, vals)
	})
	t.Run("Int slice", func(t *testing.T) {
		defer tempSet(t, key, ";1;2;77;abc")()
		vals := IntSlice(key, ";", -1)
		assert.Equal(t, []int64{-1, 1, 2, 77, -1}, vals)
	})
	t.Run("Float slice", func(t *testing.T) {
		defer tempSet(t, key, "-1-2.0-0.77-abc")()
		vals := FloatSlice(key, "-", -2.0)
		assert.Equal(t, []float64{-2.0, 1, 2, 0.77, -2.0}, vals)
	})
	t.Run("Duration slice", func(t *testing.T) {
		defer tempSet(t, key, "|1m|2s|77s|abc")()
		vals := DurationSlice(key, "|", 0)
		assert.Equal(t, []time.Duration{0, time.Minute, 2 * time.Second, 77 * time.Second, 0}, vals)
	})
}
