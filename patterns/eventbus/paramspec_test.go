package eventbus

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParamSpec(t *testing.T) {
	params := []Param{
		"A",
		nil,
		true,
		3,
		nil,
	}

	var (
		a   string
		b   bool
		c   int
		opt int
	)

	spec := ParamSpec(4,
		AssertAndStore(&a),
		nil,
		AssertAndStore(&b),
		AssertAndStore(&c),
		Optional(AssertAndStore(&opt)),
	)
	errs := spec(params)
	assert.Len(t, errs, 0)
	assert.Equal(t, "A", a)
	assert.Equal(t, true, b)
	assert.Equal(t, 3, c)
	assert.Equal(t, 0, opt)
}

func TestAnyPass(t *testing.T) {
	var (
		a string
		b int
	)
	err := AnyPass(
		AssertAndStore(&b),
		AssertAndStore(&a),
	)(1, "A value")
	assert.NoError(t, err)
	assert.Equal(t, 0, b)
	assert.Equal(t, "A value", a)
}
