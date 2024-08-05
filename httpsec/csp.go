package httpsec

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	HeaderContentSecurityPolicy = "Content-Security-Policy"
	CSPSourceSelf               = "'self'" // CSPSourceNone is the constant for the policy accepting content from the origin domain.
	CSPSourceNone               = "'none'" // CSPSourceNone is the constant for the policy accepting no content.
	CSPReportContentType        = "application/csp-report"
)

var (
	ErrContentSecurityConfig = errors.New("content security policy configuration error")
)

// CSPReport is a report that should be forwarded from the user agent when the page requests content that violates the specified CSP.
// [CSPReportHandler] can be used to receive this report.
type CSPReport struct {
	DocumentURI        string `json:"document-uri"`        // The URI of the document in which the violation occurred.
	BlockedURI         string `json:"blocked-uri"`         // The blocked resource.
	Disposition        string `json:"disposition"`         // This should always be "enforce" with this setup.
	EffectiveDirective string `json:"effective-directive"` // The specific directive that resulted in the violation.
	OriginalPolicy     string `json:"original-policy"`     // The full CSP as seen and enforced by the user agent.
	ScriptSample       string `json:"script-sample"`       // If given, includes the initial 40 bytes of the offending reference.
	StatusCode         int    `json:"status-code"`
}

type cspReportWrapper struct {
	Report CSPReport `json:"csp-report"`
}

// CSPReportHandler allows specifying a handler function for receiving CSP violation reports.
func CSPReportHandler(handler func(report CSPReport)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept", CSPReportContentType)
		defer func() {
			_ = r.Body.Close()
		}()
		var report cspReportWrapper
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			http.Error(w, "Failed to parse CSP report", 500)
			return
		}
		handler(report.Report)
	})
}

type cspConfig struct {
	ReportingEndpoint string
	DefaultSrcDomains []string
	ImageSources      []string
	MediaSources      []string
	ScriptSources     []string
	StyleSources      []string
	errors            []error
}

// CSPOption represents an option to configure content security policy behavior.
type CSPOption func(c *cspConfig)

// EnableContentSecurityPolicy allows specifying content security policies that will be applied in the [SecurityPolicies] middleware.
//
// Source: https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
func EnableContentSecurityPolicy(opts ...CSPOption) SecurityOption {
	conf := new(cspConfig)
	for _, opt := range opts {
		opt(conf)
	}
	defaultSrc := "default-src"
	specifiedDefaults := strings.Join(conf.DefaultSrcDomains, " ")
	if len(specifiedDefaults) == 0 {
		specifiedDefaults = CSPSourceSelf
	}
	defaultSrc += " " + specifiedDefaults
	sources := []string{defaultSrc}
	if len(conf.ImageSources) > 0 {
		imgSrc := "image-src " + strings.Join(conf.ImageSources, " ")
		sources = append(sources, imgSrc)
	}
	if len(conf.MediaSources) > 0 {
		mediaSrc := "media-src " + strings.Join(conf.MediaSources, " ")
		sources = append(sources, mediaSrc)
	}
	if len(conf.ScriptSources) > 0 {
		scriptSrc := "script-src " + strings.Join(conf.ScriptSources, " ")
		sources = append(sources, scriptSrc)
	}
	if len(conf.StyleSources) > 0 {
		styleSrc := "style-src " + strings.Join(conf.StyleSources, " ")
		sources = append(sources, styleSrc)
	}
	if len(conf.errors) > 0 {
		return configErrorf("%w: %s", ErrContentSecurityConfig, errors.Join(conf.errors...).Error())
	}
	return func(sec *SecurityPolicies) error {
		policy := strings.Join(sources, "; ")
		if len(conf.ReportingEndpoint) > 0 {
			sec.addReportingEndpoint("csp-endpoint", conf.ReportingEndpoint)
			policy += "; report-to csp-endpoint"
		}
		sec.headers.Set(HeaderContentSecurityPolicy, policy)
		return nil
	}
}

// DefaultSources sets the default, fallback policy for all content types.
// If a specific policy is not defined for the requested content, then this policy will apply.
//
// If no default sources are given, then the CSP will use a default source of 'self'.
// To instead specify a default of 'none', use [DefaultNone].
func DefaultSources(domains ...string) CSPOption {
	return func(c *cspConfig) {
		if err := validateCSPSourceList(domains); err != nil {
			c.errors = append(c.errors, fmt.Errorf("default sources: %w", err))
			return
		}
		c.DefaultSrcDomains = append(c.DefaultSrcDomains, domains...)
	}
}

// DefaultNone sets a default source policy where no content is allowed.
// This should be used in cases where only specific policies should be matched to the content request.
// Note that this will replace any previously set default source entries.
func DefaultNone() CSPOption {
	return func(c *cspConfig) {
		c.DefaultSrcDomains = []string{CSPSourceNone}
	}
}

// ImageSources specifies allowed domains for fetching image data.
func ImageSources(domains ...string) CSPOption {
	return func(c *cspConfig) {
		if err := validateCSPSourceList(domains); err != nil {
			c.errors = append(c.errors, fmt.Errorf("image sources: %w", err))
			return
		}
		c.ImageSources = append(c.ImageSources, domains...)
	}
}

// MediaSources specifies allowed domains for fetching media.
func MediaSources(domains ...string) CSPOption {
	return func(c *cspConfig) {
		if err := validateCSPSourceList(domains); err != nil {
			c.errors = append(c.errors, fmt.Errorf("media sources: %w", err))
			return
		}
		c.MediaSources = append(c.MediaSources, domains...)
	}
}

// ScriptSources specifies allowed domains for fetching executable scripts.
func ScriptSources(domains ...string) CSPOption {
	return func(c *cspConfig) {
		if err := validateCSPSourceList(domains); err != nil {
			c.errors = append(c.errors, fmt.Errorf("script sources: %w", err))
			return
		}
		c.ScriptSources = append(c.ScriptSources, domains...)
	}
}

// StyleSources specifies allowed domains for fetching styles.
func StyleSources(domains ...string) CSPOption {
	return func(c *cspConfig) {
		if err := validateCSPSourceList(domains); err != nil {
			c.errors = append(c.errors, fmt.Errorf("style sources: %w", err))
			return
		}
		c.StyleSources = append(c.StyleSources, domains...)
	}
}

// CSPReportingEndpoint specifies a reporting endpoint that will be called in the case of CSP violations.
// The [CSPReportHandler] may be used to easily specify a handler for these reports.
func CSPReportingEndpoint(endpoint string) CSPOption {
	return func(c *cspConfig) {
		if err := validateReportEndpoint(endpoint); err != nil {
			c.errors = append(c.errors, fmt.Errorf("reporting endpoint: %w", err))
			return
		}
		c.ReportingEndpoint = endpoint
	}
}

func validateReportEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "http":
		fallthrough
	case "https":
		return nil
	default:
		return fmt.Errorf("invalid protocol, requires either 'http' or 'https': %s", u.Scheme)
	}
}

func validateCSPSourceList(list []string) error {
	if len(list) == 0 {
		return errors.New("no sources in list, this is likely a mistake")
	}
	for i, elem := range list {
		switch elem {
		case CSPSourceNone:
			fallthrough
		case CSPSourceSelf:
			continue
		default:
			withProtocol := elem
			if !strings.HasPrefix("http", elem) {
				u, _ := url.Parse(elem)
				if len(u.Scheme) != 0 {
					// In this case there should be an invalid protocol specified.
					return fmt.Errorf("invalid protocol '%s'", u.Scheme)
				}
				withProtocol = "https://" + withProtocol
			}
			u, err := url.Parse(withProtocol)
			if err != nil {
				return fmt.Errorf("failed to parse element %d as '%s': %w", i, withProtocol, err)
			}
			if len(u.Host) != 0 && len(u.Path) != 0 {
				return fmt.Errorf("path for element %d ('%s') should be empty", i, elem)
			}
		}
	}
	return nil
}
