package env

import (
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

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
				assert.NoError(t, os.Setenv(key, tc.value))
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
	assert.NoError(t, os.Setenv(key, "true"))
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
				assert.NoError(t, os.Setenv(key, tc.value))
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
				assert.NoError(t, os.Setenv(key, tc.value))
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
				assert.NoError(t, os.Setenv(key, tc.value))
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
				assert.NoError(t, os.Setenv(key, tc.value))
			}
			assert.Equal(t, tc.expected, Duration(key, defaultVal))
		})
	}
}
