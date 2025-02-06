package httpx

import (
	"bytes"
	"context"
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
	ctx     context.Context
	client  *http.Client
	preSend []func(r *http.Request) error
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

func NewRequest(method, url string) *Request {
	r := requestInit(url)
	r.method = method
	r.client = http.DefaultClient
	return r
}

func GetRequest(u string) *Request {
	return NewRequest(http.MethodGet, u)
}

func PostRequest(u string) *Request {
	return NewRequest(http.MethodPost, u)
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
	return NewRequest(http.MethodPut, u)
}

func PatchRequest(u string) *Request {
	return NewRequest(http.MethodPatch, u)
}

func DeleteRequest(u string) *Request {
	return NewRequest(http.MethodDelete, u)
}

func (r *Request) SetHeader(header, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.headers.Set(header, value)
	return r
}

func (r *Request) WithContext(ctx context.Context) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.err != nil {
		return r
	}
	r.ctx = ctx
	return r
}

func (r *Request) AddHeader(header, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.err != nil {
		return r
	}
	r.headers.Add(header, value)
	return r
}

func (r *Request) SetQueryParams(param, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.err != nil {
		return r
	}
	q := r.u.Query()
	q.Set(param, value)
	r.u.RawQuery = q.Encode()
	return r
}

func (r *Request) AddQueryParams(param, value string) *Request {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.err != nil {
		return r
	}
	r.u.Query().Add(param, value)
	return r
}

func (r *Request) SetCookie(cookie *http.Cookie) *Request {
	if r.err != nil {
		return r
	}
	r.preSend = append(r.preSend, func(req *http.Request) error {
		if cookie == nil {
			return errors.New("nil cookie")
		}
		req.AddCookie(cookie)
		return nil
	})
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
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, r.method, r.u.String(), r.body)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers
	for _, preSend := range r.preSend {
		if preSend == nil {
			panic("nil preSend function")
		}
		if err := preSend(req); err != nil {
			return nil, err
		}
	}
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
