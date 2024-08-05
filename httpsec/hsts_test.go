package httpsec

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEnableStrictTransportSecurity(t *testing.T) {
	_, err := NewSecurityPolicies(EnableStrictTransportSecurity(5*time.Second, true))
	assert.NoError(t, err)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(5*time.Second, false))
	assert.NoError(t, err)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(0, false))
	assert.ErrorIs(t, err, ErrStrictTransportSecurity)

	_, err = NewSecurityPolicies(EnableStrictTransportSecurity(-5*time.Second, false))
	assert.ErrorIs(t, err, ErrStrictTransportSecurity)
}
