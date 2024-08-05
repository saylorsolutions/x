package httpsec

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	HeaderReportingEndpoints = "Reporting-Endpoints"
)

type SecurityPolicies struct {
	csp                cspConfig
	mw                 []func(next http.Handler) http.Handler
	reportingEndpoints map[string]string
	headers            http.Header
}

func (s *SecurityPolicies) addReportingEndpoint(key, endpoint string) {
	s.reportingEndpoints[key] = endpoint
}

type SecurityOption func(sec *SecurityPolicies) error

func configError(err error) SecurityOption {
	return func(_ *SecurityPolicies) error {
		return err
	}
}

func configErrorf(format string, args ...any) SecurityOption {
	return configError(fmt.Errorf(format, args...))
}

func NewSecurityPolicies(opts ...SecurityOption) (*SecurityPolicies, error) {
	s := new(SecurityPolicies)
	s.reportingEndpoints = map[string]string{}
	s.headers = http.Header{}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *SecurityPolicies) Middleware(next http.Handler) http.Handler {
	for i := len(s.mw) - 1; i >= 0; i-- {
		next = s.mw[i](next)
	}
	var reportingEndpoints string
	if len(s.reportingEndpoints) > 0 {
		var each []string
		for key, endpoint := range s.reportingEndpoints {
			each = append(each, fmt.Sprintf(`%s="%s"`, key, endpoint))
		}
		reportingEndpoints = strings.Join(each, ", ")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if len(reportingEndpoints) > 0 {
			w.Header().Add(HeaderReportingEndpoints, reportingEndpoints)
		}
		for header, vals := range s.headers {
			for _, val := range vals {
				w.Header().Add(header, val)
			}
		}
	})
}
