package httpsec

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_AllowEndpointAccess(t *testing.T) {
	const (
		origin = "https://example.com"
	)
	var (
		requestHandled bool
		mux            = http.NewServeMux()
	)
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		requestHandled = true
	})
	mux.HandleFunc("/testing/stuff", func(w http.ResponseWriter, r *http.Request) {
		requestHandled = true
	})
	policies, err := NewSecurityPolicies(
		EnableCORS(
			EndpointPolicy("/test", NewPolicy().
				AllowOrigin(origin).
				AllowMethods("GET", "POST").
				AllowHeader("content-type"),
			),
			EndpointPrefixPolicy("/testing", NewPolicy().
				AllowOrigin(origin).
				AllowMethods("GET", "POST").
				AllowHeader("CONTENT-TYPE"),
			),
		),
	)
	assert.NoError(t, err)
	srv := httptest.NewServer(policies.Middleware(mux))
	defer srv.Close()

	t.Run("CORS allowed", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/test", origin)
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Equal(t, origin, allowedOrigin)
		assert.Equal(t, "GET,POST", allowedMethods)
		assert.Equal(t, "Content-Type", allowedHeaders)
	})

	t.Run("CORS prefix allowed", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/testing/something", origin)
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Equal(t, origin, allowedOrigin)
		assert.Equal(t, "GET,POST", allowedMethods)
		assert.Equal(t, "Content-Type", allowedHeaders)
	})

	t.Run("CORS denied null origin", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/test", "null")
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Empty(t, allowedOrigin)
		assert.Empty(t, allowedMethods)
		assert.Empty(t, allowedHeaders)
	})

	t.Run("CORS denied no origin", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/test", "")
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Empty(t, allowedOrigin)
		assert.Empty(t, allowedMethods)
		assert.Empty(t, allowedHeaders)
	})

	t.Run("CORS denied unknown origin", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/test", "https://somethingelse.com")
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Empty(t, allowedOrigin)
		assert.Empty(t, allowedMethods)
		assert.Empty(t, allowedHeaders)
	})

	t.Run("CORS denied different endpoint", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/other", origin)
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Empty(t, allowedOrigin)
		assert.Empty(t, allowedMethods)
		assert.Empty(t, allowedHeaders)
	})
}

func TestFallbackPolicy(t *testing.T) {
	const (
		origin = "https://example.com"
	)
	var (
		requestHandled bool
		mux            = http.NewServeMux()
	)
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		requestHandled = true
	})
	mux.HandleFunc("/testing/stuff", func(w http.ResponseWriter, r *http.Request) {
		requestHandled = true
	})
	policies, err := NewSecurityPolicies(
		EnableCORS(
			EndpointPrefixPolicy("/test", NewPolicy().
				AllowAnyOrigin().
				AllowMethods("GET", "POST").
				AllowHeader("CONTENT-TYPE").
				AllowCredentials().
				MaxAge(30*24*time.Hour), // cache policy for 30 days
			),
			FallbackPolicy(NewPolicy().
				AllowOrigin(origin).
				AllowMethods("GET"),
			),
		),
	)
	assert.NoError(t, err)
	srv := httptest.NewServer(policies.Middleware(mux))
	defer srv.Close()

	t.Run("Plain GET fallback policy", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL, origin)
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Equal(t, origin, allowedOrigin)
		assert.Equal(t, "GET", allowedMethods)
		assert.Empty(t, allowedHeaders)
	})

	t.Run("Specific origin for endpoint with creds", func(t *testing.T) {
		requestHandled = false
		allowedOrigin, allowedMethods, allowedHeaders, _ := testPreflight(t, srv.URL+"/test", origin)
		assert.False(t, requestHandled, "Preflight request should not have fallen through to the handler")
		assert.Equal(t, origin, allowedOrigin)
		assert.Equal(t, "GET,POST", allowedMethods)
		assert.Equal(t, "Content-Type", allowedHeaders)
	})
}

func TestValidateCORSPolicy(t *testing.T) {
	tests := map[string]struct {
		Origin  string
		Methods []string
		Headers []string
		MaxAge  time.Duration
	}{
		"No origin": {
			Methods: []string{"GET"},
			MaxAge:  30 * 24 * time.Hour,
		},
		"No methods": {
			Origin: CORSAnyOrigin,
			MaxAge: 30 * 24 * time.Hour,
		},
		"Null origin": {
			Origin:  CORSNullOrigin,
			Methods: []string{"GET", "POST"},
			MaxAge:  30 * 24 * time.Hour,
		},
		"Invalid max age": {
			Origin:  CORSAnyOrigin,
			Methods: []string{"GET", "POST"},
			MaxAge:  0,
		},
		"Negative max age": {
			Origin:  CORSAnyOrigin,
			Methods: []string{"GET", "POST"},
			MaxAge:  -5 * time.Second,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			policy := NewPolicy()
			if len(tc.Origin) > 0 {
				policy.AllowOrigin(tc.Origin)
			}
			if len(tc.Methods) > 0 {
				policy.AllowMethods(tc.Methods...)
			}
			if len(tc.Headers) > 0 {
				policy.AllowHeader()
			}
			policy.MaxAge(tc.MaxAge)
			err := policy.validatePolicy()
			assert.Error(t, err, "Should have returned an error")
		})
	}
}

func testPreflight(t *testing.T, url, origin string) (allowedOrigin, allowedMethods, allowedHeaders, allowCredentials string) {
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	assert.NoError(t, err)
	req.Header.Set(HeaderCORSOrigin, origin)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	allowedOrigin = resp.Header.Get(HeaderCORSAllowOrigin)
	allowedMethods = resp.Header.Get(HeaderCORSAllowMethods)
	allowedHeaders = resp.Header.Get(HeaderCORSAllowHeaders)
	allowCredentials = resp.Header.Get(HeaderCORSAllowCreds)
	return
}
