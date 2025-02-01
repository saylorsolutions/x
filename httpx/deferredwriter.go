package httpx

import (
	"bytes"
	"net/http"
	"sync/atomic"
)

// DeferredWriter is a [http.ResponseWriter] implementation that holds data written to it until there's a call to [DeferredWriter.Commit].
type DeferredWriter struct {
	committed    atomic.Bool
	cached       http.ResponseWriter
	headers      http.Header
	resp         bytes.Buffer
	latestStatus int
}

func NewDeferredWriter(writer http.ResponseWriter) *DeferredWriter {
	return &DeferredWriter{
		cached:       writer,
		headers:      map[string][]string{},
		latestStatus: http.StatusOK,
	}
}

func (d *DeferredWriter) Header() http.Header {
	return d.headers
}

// Write calls are cumulative, meaning they will all contribute to the response body.
// This is consistent with the normal [http.ResponseWriter], except that order of calls do not prevent writing data to the response.
func (d *DeferredWriter) Write(data []byte) (int, error) {
	return d.resp.Write(data)
}

// WriteHeader will accept all calls, but will write the last value given to it to the underlying [http.ResponseWriter].
func (d *DeferredWriter) WriteHeader(statusCode int) {
	d.latestStatus = statusCode
}

// Commit will write all information to the underlying [http.ResponseWriter].
// Only the first call will have any effect. Subsequent calls will be ignored.
func (d *DeferredWriter) Commit() error {
	if !d.committed.CompareAndSwap(false, true) {
		return nil
	}
	for key, vals := range d.headers {
		for _, val := range vals {
			d.cached.Header().Add(key, val)
		}
	}
	if d.latestStatus != 200 {
		d.cached.WriteHeader(d.latestStatus)
	}
	_, err := d.cached.Write(d.resp.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// DeferMiddleware will create a [DeferredWriter] and pass it to wrapped handlers.
// Commit will be called after the handler returns.
func DeferMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dw := NewDeferredWriter(w)
			defer func() {
				_ = dw.Commit()
			}()
			next.ServeHTTP(dw, r)
		})
	}
}
