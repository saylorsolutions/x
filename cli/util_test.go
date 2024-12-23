package cli

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapArgs(t *testing.T) {
	var (
		a, b, c, d     string
		targets        = []*string{&a, &b, &c}
		nilTarget      = []*string{&a, &b, nil}
		tooManyTargets = []*string{&a, &b, &c, &d}
		args           = []string{"a", "b", "c"}
		tooManyArgs    = []string{"a", "b", "c", "d"}
	)

	tests := map[string]struct {
		args    []string
		targets []*string
		isError bool
	}{
		"Normal mapping": {
			args:    args,
			targets: targets,
		},
		"Not enough args": {
			args:    args[:2],
			targets: targets,
			isError: true,
		},
		"Nil args": {
			args:    nil,
			targets: targets,
			isError: true,
		},
		"Nil targets": {
			args:    args,
			targets: nil,
			isError: true,
		},
		"Not enough targets": {
			args:    args,
			targets: targets[:2],
			isError: true,
		},
		"Nil target": {
			args:    args,
			targets: nilTarget,
			isError: true,
		},
		"Too many targets": {
			args:    args,
			targets: tooManyTargets,
		},
		"Too many args": {
			args:    tooManyArgs,
			targets: targets,
		},
		"More than required": {
			args:    tooManyArgs,
			targets: tooManyTargets,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a, b, c, d = "", "", "", ""
			err := MapArgs(tc.args, 3, tc.targets...)
			if tc.isError {
				assert.ErrorIs(t, err, ErrArgMap)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, a, "a")
			assert.Equal(t, b, "b")
			assert.Equal(t, c, "c")
			if len(tc.args) == 4 && len(tc.targets) == 4 {
				assert.Equal(t, d, "d")
			}
		})
	}
}
