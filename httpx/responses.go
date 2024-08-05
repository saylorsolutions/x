package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	ErrAlreadyRead = errors.New("response body has already been read")
)

type Response struct {
	req     *http.Request
	resp    *http.Response
	mux     sync.Mutex
	hasRead bool
}

func (r *Request) Send() (*Response, int, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if r.err != nil {
		return nil, 0, r.err
	}
	req, err := http.NewRequest(r.method, r.u.String(), r.body)
	if err != nil {
		return nil, 0, err
	}
	req.Header = r.headers
	_resp := &Response{
		req: req,
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	_resp.resp = resp
	return _resp, resp.StatusCode, nil
}

func (r *Response) Close() error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.hasRead = true
	return r.resp.Body.Close()
}

func (r *Response) Body() (io.ReadCloser, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.hasRead {
		return nil, ErrAlreadyRead
	}
	r.hasRead = true
	return r.resp.Body, nil
}

func (r *Response) Bytes() ([]byte, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.hasRead {
		return nil, ErrAlreadyRead
	}
	defer func() {
		r.hasRead = true
		_ = r.resp.Body.Close()
	}()
	return io.ReadAll(r.resp.Body)
}

func (r *Response) String() (string, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.hasRead {
		return "", ErrAlreadyRead
	}
	defer func() {
		r.hasRead = true
		_ = r.resp.Body.Close()
	}()
	var buf strings.Builder
	_, err := io.Copy(&buf, r.resp.Body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ReadJSON[T any](r *Response) (*T, error) {
	var val T
	reader, err := r.Body()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	if err := json.NewDecoder(reader).Decode(&val); err != nil {
		return nil, err
	}
	return &val, nil
}
