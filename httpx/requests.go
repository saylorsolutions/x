package httpx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Request struct {
	mux     sync.RWMutex
	err     error
	method  string
	u       *url.URL
	body    io.Reader
	headers http.Header
	client  *http.Client
}

func requestInit(u string) *Request {
	_url, err := url.Parse(u)
	if err != nil {
		return &Request{err: err}
	}
	return &Request{
		u:       _url,
		headers: map[string][]string{},
		client:  http.DefaultClient,
	}
}

func GetRequest(u string) *Request {
	r := requestInit(u)
	r.method = http.MethodGet
	return r
}

func PostRequest(u string) *Request {
	r := requestInit(u)
	r.method = http.MethodPost
	return r
}

func PostFormRequest(u string, form url.Values) *Request {
	r := PostRequest(u)
	if form == nil {
		r.err = errors.New("nil form values")
		return r
	}
	r.StringBody(form.Encode())
	r.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func PutRequest(u string) *Request {
	r := requestInit(u)
	r.method = http.MethodPut
	return r
}

func PatchRequest(u string) *Request {
	r := requestInit(u)
	r.method = http.MethodPatch
	return r
}

func DeleteRequest(u string) *Request {
	r := requestInit(u)
	r.method = http.MethodDelete
	return r
}

func (r *Request) SetHeader(header, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.headers.Set(header, value)
	return r
}

func (r *Request) AddHeader(header, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.headers.Add(header, value)
	return r
}

func (r *Request) SetQueryParams(param, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	q := r.u.Query()
	q.Set(param, value)
	r.u.RawQuery = q.Encode()
	return r
}

func (r *Request) AddQueryParams(param, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.u.Query().Add(param, value)
	return r
}

func (r *Request) Body(body io.Reader) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.err != nil {
		return r
	}
	r.body = body
	return r
}

func (r *Request) StringBody(body string) *Request {
	r.Body(strings.NewReader(body))
	return r
}

func (r *Request) BytesBody(body []byte) *Request {
	r.Body(bytes.NewReader(body))
	return r
}

func (r *Request) JSONBody(body any) *Request {
	data, err := json.Marshal(body)
	if err != nil {
		r.mux.Lock()
		defer r.mux.Unlock()
		r.err = err
		return r
	}
	r.SetHeader("Content-Type", "application/json")
	r.BytesBody(data)
	return r
}

func (r *Request) StdRequest() (*http.Request, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if r.err != nil {
		return nil, r.err
	}
	req, err := http.NewRequest(r.method, r.u.String(), r.body)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers
	return req, nil
}

func (r *Request) BasicAuth(user, pass string) *Request {
	authStr := base64.URLEncoding.EncodeToString([]byte(user + ":" + pass))
	r.SetHeader("Authorization", "Basic "+authStr)
	return r
}

func (r *Request) BearerAuth(token string) *Request {
	r.SetHeader("Authorization", "Bearer "+token)
	return r
}
