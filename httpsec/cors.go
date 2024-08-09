package httpsec

import (
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

const (
	HeaderCORSOrigin       = "Origin"
	HeaderCORSVary         = "Vary"
	HeaderCORSAllowOrigin  = "Access-Control-Allow-Origin"
	HeaderCORSAllowMethods = "Access-Control-Allow-Methods"
	HeaderCORSAllowHeaders = "Access-Control-Allow-Headers"
	HeaderCORSAllowCreds   = "Access-Control-Allow-Credentials"
	HeaderCORSMaxAge       = "Access-Control-Max-Age"

	CORSAnyOrigin  = "*"
	CORSNullOrigin = "null"
)

var (
	ErrCORSPolicy    = errors.New("CORS policy error")
	ErrCORSNoOrigin  = errors.New("no allowed origins specified")
	ErrCORSNoMethods = errors.New("no allowed methods specified")
)

type CORSPolicy struct {
	allowedMethods   stringSet
	allowedHeaders   stringSet
	allowedOrigins   stringSet
	allowCredentials bool
	maxAge           time.Duration
	err              error
}

func NewPolicy() *CORSPolicy {
	return &CORSPolicy{
		allowedMethods: stringSet{},
		allowedHeaders: stringSet{},
		allowedOrigins: stringSet{},
		maxAge:         86400 * time.Second, // Default to 24 hours.
	}
}

type corsMapping map[string]CORSPolicy

func (m corsMapping) matchPrefix(endpoint string) (CORSPolicy, bool) {
	var mt CORSPolicy
	for prefix, policy := range m {
		if strings.HasPrefix(endpoint, prefix) {
			return policy, true
		}
	}
	return mt, false
}

type corsConfig struct {
	fallbackPolicy   *CORSPolicy
	endpointPolicies corsMapping
	errs             []error
	prefixPolicies   corsMapping
}

type CORSOption func(c *corsConfig)

// EnableCORS sets up Cross Origin Resource Sharing (CORS) in the [SecurityPolicies].
//
// CORS is intended to explicitly and safely allow cross-origin requests from the browser.
// A web server by default doesn't allow cross-origin requests because no Allow headers will be sent.
// So this is provided to explicitly allow such requests in cases where it makes sense.
//
// This implementation is somewhat opinionated.
// This will not allow a null origin to be accepted, because it enables a few classes of vulnerabilities.
// It also does not use wildcard prefixed/suffixed origins.
// These can usually be easily exploited despite an honest attempt to limit exposure.
// This does allow for accepting traffic from any origin (*), but should ONLY be used when truly ANY site should be able to access the content.
//
// At the end of the day, the premise of CORS relies entirely on the correct behavior of the browser, which cannot be relied upon as any kind of silver bullet solution (defense in depth).
//
// Source: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
func EnableCORS(options ...CORSOption) SecurityOption {
	conf := &corsConfig{
		endpointPolicies: map[string]CORSPolicy{},
		prefixPolicies:   map[string]CORSPolicy{},
	}
	for _, opt := range options {
		opt(conf)
	}
	if len(conf.errs) > 0 {
		return configErrorf("%w: %s", ErrCORSPolicy, errors.Join(conf.errs...))
	}
	return func(sec *SecurityPolicies) error {
		sec.mw = append(sec.mw, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodOptions {
					// This is a preflight.
					// This middleware will take control of these requests.
					conf.grantAllowHeaders(w, r)
					return
				}
				next.ServeHTTP(w, r)
				// Send minimal allow headers otherwise.
				conf.grantAllowHeaders(w, r)
			})
		})
		return nil
	}
}

// FallbackPolicy specifies the CORS allowances when no endpoint policy is found.
// If this is not specified, then the default will be to not allow cross-origin resource sharing.
func FallbackPolicy(policy *CORSPolicy) CORSOption {
	return func(c *corsConfig) {
		if err := policy.validatePolicy(); err != nil {
			c.errs = append(c.errs, err)
			return
		}
		c.fallbackPolicy = policy
	}
}

// EndpointPolicy specifies the CORS allowances for the given endpoint.
// This will use an exact-match criteria to determine if the policy applies to the given request.
func EndpointPolicy(endpoint string, policy *CORSPolicy) CORSOption {
	return func(c *corsConfig) {
		if len(endpoint) == 0 {
			c.errs = append(c.errs, errors.New("attempted to set endpoint policy with no endpoint"))
		}
		if err := policy.validatePolicy(); err != nil {
			c.errs = append(c.errs, err)
			return
		}
		c.endpointPolicies[endpoint] = *policy
	}
}

// EndpointPrefixPolicy specifies the CORS allowances for endpoints with the given path prefix.
// This will use a starts-with criteria to determine if the policy applies to the given request.
func EndpointPrefixPolicy(prefix string, policy *CORSPolicy) CORSOption {
	return func(c *corsConfig) {
		if len(prefix) == 0 {
			c.errs = append(c.errs, errors.New("attempted to set endpoint policy with no endpoint"))
		}
		if err := policy.validatePolicy(); err != nil {
			c.errs = append(c.errs, err)
			return
		}
		c.prefixPolicies[prefix] = *policy
	}
}

func (p *CORSPolicy) validatePolicy() error {
	if p.err != nil {
		return p.err
	}
	if len(p.allowedOrigins) == 0 {
		return ErrCORSNoOrigin
	}
	if len(p.allowedMethods) == 0 {
		return ErrCORSNoMethods
	}
	return nil
}

// MaxAge sets the max cache time for this policy.
// A policy's default max age without calling this method is 24 hours.
func (p *CORSPolicy) MaxAge(ttl time.Duration) *CORSPolicy {
	if ttl <= 0 {
		p.err = errors.New("max-age is <= 0")
		return p
	}
	p.maxAge = ttl
	return p
}

// AllowMethods is a convenience method that enables specifying multiple allowed request methods for this policy.
func (p *CORSPolicy) AllowMethods(methods ...string) *CORSPolicy {
	for _, method := range methods {
		switch strings.TrimSpace(strings.ToUpper(method)) {
		case "GET":
			p.AllowGet()
		case "POST":
			p.AllowPost()
		case "PUT":
			p.AllowPut()
		case "PATCH":
			p.AllowPatch()
		case "DELETE":
			p.AllowDelete()
		}
	}
	return p
}

// AllowGet allows the GET method for this policy.
func (p *CORSPolicy) AllowGet() *CORSPolicy {
	p.allowedMethods.add("GET")
	return p
}

// AllowPost allows the POST method for this policy.
func (p *CORSPolicy) AllowPost() *CORSPolicy {
	p.allowedMethods.add("POST")
	return p
}

// AllowPut allows the PUT method for this policy.
func (p *CORSPolicy) AllowPut() *CORSPolicy {
	p.allowedMethods.add("PUT")
	return p
}

// AllowPatch allows the PATCH method for this policy.
func (p *CORSPolicy) AllowPatch() *CORSPolicy {
	p.allowedMethods.add("PATCH")
	return p
}

// AllowDelete allows the DELETE method for this policy.
func (p *CORSPolicy) AllowDelete() *CORSPolicy {
	p.allowedMethods.add("DELETE")
	return p
}

// AllowHeader allows the given header to be sent.
func (p *CORSPolicy) AllowHeader(headers ...string) *CORSPolicy {
	for _, header := range headers {
		header = textproto.CanonicalMIMEHeaderKey(header)
		p.allowedHeaders.add(header)
	}
	return p
}

// AllowAnyOrigin sets this policy's origin allow list to be *, allowing any origin.
func (p *CORSPolicy) AllowAnyOrigin() *CORSPolicy {
	p.allowedOrigins = stringSet{CORSAnyOrigin: true}
	return p
}

// AllowOrigin allows access to the given origin for this policy.
// Calling this method will override a previous call to [CORSPolicy.AllowAnyOrigin].
func (p *CORSPolicy) AllowOrigin(origins ...string) *CORSPolicy {
	for _, origin := range origins {
		if origin == CORSAnyOrigin {
			return p.AllowAnyOrigin()
		}
		u, err := url.Parse(origin)
		if err != nil || len(u.Scheme) == 0 {
			// Not a valid origin
			p.err = fmt.Errorf("invalid origin '%s': %w", origin, err)
			return p
		}
		origin = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		if port := u.Port(); len(port) > 0 {
			origin += ":" + port
		}
		p.allowedOrigins.remove(CORSAnyOrigin)
		p.allowedOrigins.add(origin)
	}
	return p
}

// AllowCredentials sets the credentials flag when responding to OPTIONS preflight requests.
func (p *CORSPolicy) AllowCredentials() *CORSPolicy {
	p.allowCredentials = true
	return p
}
