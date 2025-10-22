package regexpx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVarPattern_Define(t *testing.T) {
	vp := NewVarPattern()
	input := "    som   e tex   t is     here   "
	require.NoError(t, vp.Define("three", `\s{3}`))
	require.NoError(t, vp.Define("four", `\s{4}`))
	pat := vp.MustCompile(`([[:$four:]]|[[:$three:]])`)
	result := pat.ReplaceAllString(input, "")
	assert.Equal(t, "some text is here", result)
}

func TestVarPattern_Define_BadName(t *testing.T) {
	inputs := []string{
		"A",
		"123",
		"fourTfour",
		"se7en",
		"^_^",
		"o7",
		"no space",
	}
	vp := NewVarPattern()
	for _, input := range inputs {
		t.Run(fmt.Sprintf("Trying pattern name '%s'", input), func(t *testing.T) {
			assert.Error(t, vp.Define(input, `.*`))
		})
	}
}

func TestVarPattern_Compile_Undefined(t *testing.T) {
	vp := NewVarPattern()
	_, err := vp.Compile(`[[:$something:]]`)
	assert.ErrorIs(t, err, ErrUndefinedRef)
}

func TestVarPattern_Compile_InvalidPattern(t *testing.T) {
	const invalidPattern = `\b[M]\w+\`
	vp := NewVarPattern()
	_, err := vp.Compile(invalidPattern)
	assert.Error(t, err)
	assert.Error(t, vp.Define("name", invalidPattern))
	assert.Panics(t, func() {
		vp.MustCompile(invalidPattern)
	})
	assert.Panics(t, func() {
		vp.MustDefine("name", invalidPattern)
	})
	require.NoError(t, vp.Define("name", `\b[M]\w+`))
	_, err = vp.Compile(`[[:$name:]]\`)
	assert.Error(t, err)
}

func TestVarPattern_Compile_NoNameToReplace(t *testing.T) {
	vp := NewVarPattern()
	_, err := vp.Compile(`Won't be replaced: [[:$:]]`)
	t.Log(err)
	assert.Error(t, err)
}
