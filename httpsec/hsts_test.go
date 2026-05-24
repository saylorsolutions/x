package httpsec

import (
	"github.com/stretchr/testify/require"

	"testing"
	"time"
)

func TestEnableStrictTransportSecurity(t *testing.T) {
	_, err := NewSecurityPolicies(EnableStrictTransportSecurity(5*time.Second, true))
	require.NoError(t, err)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(5*time.Second, false))
	require.NoError(t, err)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(0, false))
	require.ErrorIs(t, err, ErrStrictTransportSecurity)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(-5*time.Second, false))
	require.ErrorIs(t, err, ErrStrictTransportSecurity)
}
