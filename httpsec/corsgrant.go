package httpsec

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (c *corsConfig) grantAllowHeaders(w http.ResponseWriter, r *http.Request) {
	policy, ok := c.endpointPolicies[r.URL.Path]
	if !ok {
		if policy, ok = c.prefixPolicies.matchPrefix(r.URL.Path); !ok {
			if c.fallbackPolicy == nil {
				// No policy matches this endpoint
				if r.Method == http.MethodOptions {
					// If this is a preflight then inform the client that there is no resource here.
					w.WriteHeader(404)
				}
				return
			}
			policy = *c.fallbackPolicy
		}
	}
	reqOrigin := r.Header.Get(HeaderCORSOrigin)
	if len(reqOrigin) == 0 || reqOrigin == CORSNullOrigin {
		// Only respond to requests with Origin header and explicitly disallow null Origin.
		return
	}
	var (
		respOrigin string
		varyOrigin bool
	)
	switch {
	case policy.allowedOrigins.has(CORSAnyOrigin):
		// All origins are allowed, output policy.
		if policy.allowCredentials {
			// Credentials are allowed, just output the given origin.
			respOrigin = reqOrigin
			varyOrigin = true
		} else {
			// Credentials are not allowed, we can output *
			respOrigin = CORSAnyOrigin
		}
	case policy.allowedOrigins.has(reqOrigin):
		// This origin is in the set of allowed origins, output policy.
		respOrigin = reqOrigin
		if len(policy.allowedOrigins) > 1 {
			varyOrigin = true
		}
	default:
		// This origin isn't trusted.
		// CORS denies by default. So by not sending any allowed headers, the request fails in preflight.
		return
	}
	if r.Method == http.MethodOptions {
		// Send the other allow headers for preflight.
		respMethod := strings.Join(policy.allowedMethods.slice(), ",")
		w.Header().Set(HeaderCORSAllowMethods, respMethod)
		respHeaders := strings.Join(policy.allowedHeaders.slice(), ",")
		if len(respHeaders) > 0 {
			// Allowing headers isn't actually required.
			w.Header().Set(HeaderCORSAllowHeaders, respHeaders)
		}
		maxAgeSeconds := strconv.Itoa(int(policy.maxAge.Round(time.Second).Seconds()))
		w.Header().Set(HeaderCORSMaxAge, maxAgeSeconds)
	}
	w.Header().Set(HeaderCORSAllowOrigin, respOrigin)
	if varyOrigin {
		w.Header().Set(HeaderCORSVary, HeaderCORSOrigin)
	}
	if policy.allowCredentials {
		w.Header().Set(HeaderCORSAllowCreds, "true")
	}
}
