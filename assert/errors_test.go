package assert

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCollector_Unwrap(t *testing.T) {
	var (
		ErrA = errors.New("A")
		ErrB = errors.New("B")
		err  = CollectErrors().Add(ErrA).Add(ErrB).Result()
		as   = new(Collector)
	)

	require.NotNil(t, err)
	assert.ErrorIs(t, err, ErrA)
	assert.ErrorIs(t, err, ErrB)
	assert.ErrorAs(t, err, &as)
}

func TestCollector_Error(t *testing.T) {
	var (
		ErrA = errors.New("A")
		ErrB = errors.New("B")
		err  = CollectErrors(" ").Add(ErrA).Add(ErrB).AddString("C").Result()
	)
	require.NotNil(t, err)
	assert.Equal(t, "A B C", err.Error())
}
