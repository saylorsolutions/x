package httpx

import (
	"errors"
	"fmt"
	"time"
)

const (
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
)

var (
	ErrStrictTransportSecurity = errors.New("strict transport security")
)

// EnableStrictTransportSecurity enables sending the HTTP Strict Transport Security (HSTS) header to ensure that connections are established with TLS.
// This helps to prevent man-in-the-middle (MITM) attacks against a previously visited web server, because the user agent will cache this header for max-age seconds.
// Of course, enabling this necessitates serving content using TLS.
//
// Note that 'preload' is currently not part of the specification, and thus not supported.
//
// Source: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
func EnableStrictTransportSecurity(maxAge time.Duration, includeSubdomains bool) SecurityOption {
	rounded := maxAge.Round(time.Second)
	if rounded <= 0 {
		return configErrorf("%w: max age (%d) <= 0 seconds", ErrStrictTransportSecurity, maxAge)
	}
	seconds := int64(rounded.Seconds())
	val := fmt.Sprintf("max-age=%d", seconds)
	if includeSubdomains {
		val += "; includeSubDomains"
	}
	return func(sec *SecurityPolicies) error {
		sec.headers.Set(HeaderStrictTransportSecurity, val)
		return nil
	}
}
