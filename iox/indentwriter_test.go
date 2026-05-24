package iox

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndentWriter_Write(t *testing.T) {
	tests := map[string]struct {
		expected string
		before   string
		indented string
		after    string
	}{
		"Normalized line endings": {
			before:   "line1\nline2\r\nline3\n",
			expected: "line1\nline2\nline3\n",
		},
		"Structured output": {
			before:   "if a == b {\r\n",
			indented: "fmt.Println(\"a == b\")\r\nreturn 0\n",
			after:    "}\r\n",
			expected: "if a == b {\n\tfmt.Println(\"a == b\")\n\treturn 0\n}\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				output bytes.Buffer
				iw     = IndentWriter{
					Out: &output,
				}
			)
			if len(tc.before) > 0 {
				_, err := iw.Write([]byte(tc.before))
				require.NoError(t, err)
			}
			if len(tc.indented) > 0 {
				iw.Indent()
				_, err := iw.Write([]byte(tc.indented))
				require.NoError(t, err)
				_, err = iw.Outdent()
				require.NoError(t, err)
			}
			if len(tc.after) > 0 {
				_, err := iw.Write([]byte(tc.after))
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expected, output.String())
		})
	}
}
