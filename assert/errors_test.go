package assert

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_Unwrap(t *testing.T) {
	var (
		ErrA = errors.New("A")
		ErrB = errors.New("B")
		err  = CollectErrors().Add(ErrA).Add(ErrB).Result()
		as   = new(Collector)
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrA)
	require.ErrorIs(t, err, ErrB)
	require.ErrorAs(t, err, &as)
}

func TestCollector_Error(t *testing.T) {
	var (
		ErrA = errors.New("A")
		ErrB = errors.New("B")
		err  = CollectErrors(" ").Add(ErrA).Add(ErrB).AddString("C").Result()
	)
	require.Error(t, err)
	assert.Equal(t, "A B C", err.Error())
}
