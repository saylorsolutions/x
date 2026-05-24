package httpx

import (
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed static
	staticAssets embed.FS
)

func TestEmbeddedHandler(t *testing.T) {
	expected, err := staticAssets.ReadFile("static/test.svg")
	require.NoError(t, err)
	require.NotEmpty(t, expected)

	mux := http.NewServeMux()
	mux.Handle("GET /something/", EmbeddedHandler(staticAssets, "/something", "/static"))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, status, err := GetRequest(fmt.Sprintf("%s/something/test.svg", srv.URL)).Send()
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.NotNil(t, resp)
	contentType, ok := resp.GetHeader("Content-Type")
	assert.True(t, ok, "Should have sent Content-Type header")
	assert.Equal(t, "image/svg+xml", contentType)

	data, err := resp.Bytes()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, expected, data)

	_, status, err = GetRequest(fmt.Sprintf("%s/something/else.css", srv.URL)).Send()
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}
