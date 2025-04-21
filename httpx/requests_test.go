package httpx

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestJSONType struct {
	Name string `json:"name"`
}

func TestRequest(t *testing.T) {
	var (
		handledRequest bool
		hasQueryParam  bool
		hasFormParam   bool
		hasJSONData    bool
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		handledRequest = true
		if r.URL.Query().Has("key") {
			hasQueryParam = true
		}
		if val := r.FormValue("name"); val == "bob" {
			hasFormParam = true
		}
		if r.Header.Get("Content-Type") == "application/json" {
			var obj TestJSONType
			if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if obj.Name == "bob" {
				hasJSONData = true
			}
		}
		_, _ = w.Write([]byte("all good!"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Run("Get request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := GetRequest(srv.URL+"/test").SetQueryParams("key", "value").Send()
		require.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.True(t, hasQueryParam, "Should have had the query parameter")
		assert.False(t, hasFormParam, "Should not have had the form parameter")
		assert.False(t, hasJSONData, "Should not have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})

	t.Run("Post request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := PostRequest(srv.URL + "/test").Send()
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.False(t, hasQueryParam, "Should not have had the query parameter")
		assert.False(t, hasFormParam, "Should not have had the form parameter")
		assert.False(t, hasJSONData, "Should not have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})

	t.Run("Post form request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := PostFormRequest(srv.URL+"/test", map[string][]string{
			"name": {"bob"},
		}).Send()
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.False(t, hasQueryParam, "Should not have had the query parameter")
		assert.True(t, hasFormParam, "Should have had the form parameter")
		assert.False(t, hasJSONData, "Should not have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})

	t.Run("Put request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := PutRequest(srv.URL + "/test").JSONBody(TestJSONType{Name: "bob"}).Send()
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.False(t, hasQueryParam, "Should not have had the query parameter")
		assert.False(t, hasFormParam, "Should not have had the form parameter")
		assert.True(t, hasJSONData, "Should have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})

	t.Run("Patch request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := PatchRequest(srv.URL + "/test").JSONBody(TestJSONType{Name: "bob"}).Send()
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.False(t, hasQueryParam, "Should not have had the query parameter")
		assert.False(t, hasFormParam, "Should not have had the form parameter")
		assert.True(t, hasJSONData, "Should have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})

	t.Run("Delete request", func(t *testing.T) {
		handledRequest = false
		hasQueryParam = false
		hasFormParam = false
		hasJSONData = false
		resp, status, err := DeleteRequest(srv.URL + "/test").Send()
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 200, status)
		assert.True(t, handledRequest, "Should have handled request")
		assert.False(t, hasQueryParam, "Should not have had the query parameter")
		assert.False(t, hasFormParam, "Should not have had the form parameter")
		assert.False(t, hasJSONData, "Should not have JSON data")
		str, err := resp.String()
		assert.NoError(t, err)
		assert.Equal(t, "all good!", str)
	})
}

type bufCloser struct {
	io.Reader
	closeCalled bool
}

func (c *bufCloser) Close() error {
	c.closeCalled = true
	return nil
}

func TestReadJSON(t *testing.T) {
	body := &bufCloser{Reader: strings.NewReader(`{"name":"bob"}`)}
	resp := &Response{
		resp: &http.Response{
			Body: body,
		},
	}
	val, err := ReadJSON[map[string]any](resp)
	assert.NoError(t, err)
	assert.Equal(t, "bob", (*val)["name"].(string))
	assert.True(t, body.closeCalled, "Close wasn't called on response body")
}

func TestRequestAuth(t *testing.T) {
	var (
		basicAuth, bearerAuth bool
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); ok {
			if user == "jamesbaxter" && pass == "neigh" {
				basicAuth = true
				return
			}
		}
		if authHeader := r.Header.Get("Authorization"); len(authHeader) > 0 {
			prefix, value, ok := strings.Cut(authHeader, " ")
			if ok {
				if prefix == "Bearer" && value == "12345" {
					bearerAuth = true
					return
				}
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Run("Basic auth", func(t *testing.T) {
		basicAuth = false
		bearerAuth = false
		resp, status, err := PostRequest(srv.URL+"/test").BasicAuth("jamesbaxter", "neigh").Send()
		assert.Equal(t, 200, status)
		assert.NoError(t, err)
		assert.NoError(t, resp.Close())
		assert.True(t, basicAuth, "Should have capture basic auth credentials")
		assert.False(t, bearerAuth, "Should not have capture bearer auth credentials")
	})

	t.Run("Bearer auth", func(t *testing.T) {
		basicAuth = false
		bearerAuth = false
		resp, status, err := PostRequest(srv.URL + "/test").BearerAuth("12345").Send()
		assert.Equal(t, 200, status)
		assert.NoError(t, err)
		assert.NoError(t, resp.Close())
		assert.False(t, basicAuth, "Should not have capture basic auth credentials")
		assert.True(t, bearerAuth, "Should have capture bearer auth credentials")
	})
}

func TestRequest_SetCookie(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /cookie", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "cookie value", cookies[0].Value)
		assert.Equal(t, "TestCookie", cookies[0].Name)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, status, err := GetRequest(srv.URL + "/cookie").SetCookie(&http.Cookie{
		Name:     "TestCookie",
		Value:    "cookie value",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}).Send()
	assert.NoError(t, err)
	assert.Equal(t, 200, status)
}

func TestRequest_JSONBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /json", func(w http.ResponseWriter, r *http.Request) {
		req := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error("Unexpected error decoding JSON:", err)
		}
		assert.Equal(t, "payload", req["payload"])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, status, err := PostRequest(srv.URL + "/json").
		JSONBody(map[string]any{
			"payload": "payload",
		}).Send()
	_ = resp.Close()
	assert.NoError(t, err)
	assert.Equal(t, 200, status)
}
