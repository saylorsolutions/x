package httpx

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnableContentSecurityPolicy(t *testing.T) {
	tests := map[string]struct {
		opts           []CSPOption
		expectedPolicy string
		expectedReport string
	}{
		"Default configuration": {
			expectedPolicy: "default-src 'self'",
		},
		"Default none": {
			opts: []CSPOption{
				DefaultNone(),
			},
			expectedPolicy: "default-src 'none'",
		},
		"Some defaults": {
			opts: []CSPOption{
				DefaultSources("example.com"),
			},
			expectedPolicy: "default-src example.com",
		},
		"Reporting endpoint": {
			opts: []CSPOption{
				CSPReportingEndpoint("https://example.com/csp-report"),
			},
			expectedReport: `csp-endpoint="https://example.com/csp-report"`,
			expectedPolicy: "default-src 'self'; report-to csp-endpoint",
		},
		"Image policy": {
			opts: []CSPOption{
				ImageSources("example.com", "*.example.com"),
			},
			expectedPolicy: "default-src 'self'; image-src example.com *.example.com",
		},
		"Media policy": {
			opts: []CSPOption{
				MediaSources("example.com", "*.example.com"),
			},
			expectedPolicy: "default-src 'self'; media-src example.com *.example.com",
		},
		"Style policy": {
			opts: []CSPOption{
				StyleSources("example.com", "*.example.com"),
			},
			expectedPolicy: "default-src 'self'; style-src example.com *.example.com",
		},
		"Script policy": {
			opts: []CSPOption{
				ScriptSources("example.com", "*.example.com"),
			},
			expectedPolicy: "default-src 'self'; script-src example.com *.example.com",
		},
		"Multiple policies": {
			opts: []CSPOption{
				ScriptSources("example.com", "*.example.com"),
				StyleSources("*"),
				MediaSources(CSPSourceNone),
				CSPReportingEndpoint("https://example.com/csp-report"),
			},
			expectedPolicy: "default-src 'self'; media-src 'none'; script-src example.com *.example.com; style-src *; report-to csp-endpoint",
			expectedReport: `csp-endpoint="https://example.com/csp-report"`,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			called := false
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// handle all requests
				called = true
			})
			sec, err := NewSecurityPolicies(EnableContentSecurityPolicy(tc.opts...))
			require.NoError(t, err)
			srv := httptest.NewServer(sec.Middleware(mux))
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			assert.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.True(t, called)
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, tc.expectedPolicy, resp.Header.Get(HeaderContentSecurityPolicy))
			if len(tc.expectedReport) > 0 {
				assert.Equal(t, tc.expectedReport, resp.Header.Get(HeaderReportingEndpoints))
			} else {
				_, ok := resp.Header[HeaderReportingEndpoints]
				assert.False(t, ok, "Should not have specified a reporting endpoint")
			}
		})
	}
}

func TestEnableContentSecurityPolicy_Neg(t *testing.T) {
	tests := map[string]CSPOption{
		"Empty defaults":                        DefaultSources(),
		"Default with path":                     DefaultSources("some.domain/with-path"),
		"Empty media":                           MediaSources(),
		"Invalid media":                         MediaSources("ftp://blah.com"),
		"Media with path":                       MediaSources("some.domain/with-path"),
		"Empty image":                           ImageSources(),
		"Invalid image":                         ImageSources("ftp://blah.com"),
		"Image with path":                       ImageSources("some.domain/with-path"),
		"Empty script":                          ScriptSources(),
		"Invalid script":                        ScriptSources("ftp://blah.com"),
		"Script with path":                      ScriptSources("some.domain/with-path"),
		"Empty style":                           StyleSources(),
		"Invalid style":                         StyleSources("ftp://blah.com"),
		"Style with path":                       StyleSources("some.domain/with-path"),
		"Report endpoint with no protocol":      CSPReportingEndpoint("some.domain"),
		"Report endpoint with invalid protocol": CSPReportingEndpoint("ftp://some.domain"),
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			_, err := NewSecurityPolicies(EnableContentSecurityPolicy(tc))
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrContentSecurityConfig)
		})
	}
}

func TestCSPReportHandler(t *testing.T) {
	var (
		reportReceived bool
		report         CSPReport
	)
	mux := http.NewServeMux()
	mux.Handle("/", CSPReportHandler(func(_report CSPReport) {
		reportReceived = true
		report = _report
	}))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	givenReport := CSPReport{
		DocumentURI:        "https://example.com",
		BlockedURI:         "https://malicious.com",
		Disposition:        "enforce",
		EffectiveDirective: "default-src 'self'",
		OriginalPolicy:     "default-src 'self'",
		ScriptSample:       `<script src="https://malicious.com/bomb.`,
		StatusCode:         200,
	}
	body, err := json.Marshal(map[string]any{
		"csp-report": givenReport,
	})
	require.NoError(t, err)
	resp, err := http.Post(srv.URL, CSPReportContentType, bytes.NewReader(body))
	assert.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, reportReceived, "Didn't receive report!")
	assert.Equal(t, givenReport, report)
}
