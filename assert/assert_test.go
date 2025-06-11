package assert_test

import (
	"github.com/saylorsolutions/x/assert"
	"testing"
)

func TestNotEmpty(t *testing.T) {
	var (
		slice []byte
		str   string
		mapp  map[string]bool
		// Sticking with the most common types as an example, but there are others supported.
	)
	tests := map[string]any{
		"Empty slice":  slice,
		"Empty string": str,
		"Empty map":    mapp,
	}
	for name, val := range tests {
		val := val
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Log(r)
				} else {
					t.Failed()
				}
			}()
			assert.NotEmpty(name, val)
		})
	}
}

func TestTrue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("Should not have panicked:", r)
		}
	}()
	assert.True("true", true)
	assert.TrueFunc("returns true", func() bool {
		return true
	})
}

func TestDisable(t *testing.T) {
	assert.Disable()
	t.Cleanup(func() {
		assert.Enable()
	})
	assert.True("false", false)
	assert.TrueFunc("also false", func() bool {
		return false
	})
	assert.NotEmpty("empty string", "")
}
