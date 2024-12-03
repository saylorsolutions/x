package httpx

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeferredResponseWriter_Commit(t *testing.T) {
	t.Run("Status code then header", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.Header().Set("X-SOME-HEADER", "value")
		}
		status, body, headers := testUseWriter(t, handler)
		assert.Equal(t, http.StatusAccepted, status)
		assert.Empty(t, body)
		assert.Equal(t, "value", headers.Get("X-SOME-HEADER"))
		status, body, headers = testWithoutWriter(t, handler)
		assert.Equal(t, http.StatusAccepted, status)
		assert.Empty(t, body)
		assert.Empty(t, headers.Get("X-SOME-HEADER"))
	})
	t.Run("Body then status code and header", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("body"))
			w.WriteHeader(http.StatusAccepted)
			w.Header().Set("X-SOME-HEADER", "value")
		}
		status, body, headers := testUseWriter(t, handler)
		assert.Equal(t, http.StatusAccepted, status)
		assert.Equal(t, []byte("body"), body)
		assert.Equal(t, "value", headers.Get("X-SOME-HEADER"))
		status, body, headers = testWithoutWriter(t, handler)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, []byte("body"), body)
		assert.Empty(t, headers.Get("X-SOME-HEADER"))
	})
	t.Run("No action", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {}
		status, body, headers := testUseWriter(t, handler)
		assert.Equal(t, http.StatusOK, status)
		assert.Empty(t, body)
		assert.Empty(t, headers.Get("X-SOME-HEADER"))
		status, body, headers = testWithoutWriter(t, handler)
		assert.Equal(t, http.StatusOK, status)
		assert.Empty(t, body)
		assert.Empty(t, headers.Get("X-SOME-HEADER"))
	})
	t.Run("Header, body, then status", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("key", "value")
			_, err := w.Write([]byte("body"))
			assert.NoError(t, err)
			w.WriteHeader(202)
		}
		status, body, headers := testUseWriter(t, handler)
		assert.Equal(t, 202, status)
		assert.Equal(t, []byte("body"), body)
		assert.Equal(t, "value", headers.Get("key"))
		status, body, headers = testWithoutWriter(t, handler)
		assert.Equal(t, 200, status)
		assert.Equal(t, []byte("body"), body)
		assert.Equal(t, "value", headers.Get("key"))
	})
	t.Run("Set cookie", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:  "test-cookie",
				Value: "cookie-value",
			})
			http.SetCookie(w, &http.Cookie{
				Name:  "other-cookie",
				Value: "cookie-value",
			})
		}
		status, body, headers := testUseWriter(t, handler)
		assert.Equal(t, []string{"test-cookie=cookie-value", "other-cookie=cookie-value"}, headers.Values("Set-Cookie"))
		assert.Equal(t, 200, status)
		assert.Empty(t, body)
	})
}

func testUseWriter(t *testing.T, handler http.HandlerFunc) (int, []byte, http.Header) {
	wrapped := func(w http.ResponseWriter, r *http.Request) {
		dw := NewDeferredWriter(w)
		handler.ServeHTTP(dw, r)
		if err := dw.Commit(); err != nil {
			t.Error("Got an error from committing the deferred writer")
		}
	}
	return testWithoutWriter(t, wrapped)
}

func testWithoutWriter(t *testing.T, handler http.HandlerFunc) (int, []byte, http.Header) {
	mux := http.NewServeMux()
	mux.Handle("/test", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	if err != nil {
		t.Error("Failed to get server response:", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var body []byte
	if resp.ContentLength > 0 {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Error("Error reading response body:", err)
		}
	}
	return resp.StatusCode, body, resp.Header
}
