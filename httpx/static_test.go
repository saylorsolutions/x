package httpx

import (
	"embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	//go:embed static
	staticAssets embed.FS
)

func TestEmbeddedHandler(t *testing.T) {
	expected, err := staticAssets.ReadFile("static/test.svg")
	require.NoError(t, err)
	require.True(t, len(expected) > 0)

	mux := http.NewServeMux()
	mux.Handle("GET /something/", EmbeddedHandler(staticAssets, "/something", "/static"))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, status, err := GetRequest(fmt.Sprintf("%s/something/test.svg", srv.URL)).Send()
	assert.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.NotNil(t, resp)
	contentType, ok := resp.GetHeader("Content-Type")
	assert.True(t, ok, "Should have sent Content-Type header")
	assert.Equal(t, "image/svg+xml", contentType)

	data, err := resp.Bytes()
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, expected, data)

	_, status, err = GetRequest(fmt.Sprintf("%s/something/else.css", srv.URL)).Send()
	assert.NoError(t, err)
	assert.Equal(t, 404, status)
}
