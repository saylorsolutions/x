package httpx

import (
	"embed"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ContentHandler(contentType string, data io.Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = io.Copy(w, data)
	}
}

// CSSHandler will set the correct content type for CSS data, and copy the reader to the response.
func CSSHandler(data io.Reader) http.HandlerFunc {
	return ContentHandler("text/css", data)
}

// JSHandler will set the correct content type for JS data, and copy the reader to the response.
func JSHandler(data io.Reader) http.HandlerFunc {
	return ContentHandler("text/javascript", data)
}

// HTMLHandler will set the correct content type for HTTP data, and copy the reader to the response.
func HTMLHandler(data io.Reader) http.HandlerFunc {
	return ContentHandler("text/html", data)
}

// ExtensionMapping is used to match common file extensions to a MIME type.
// This can be customized by the user to include other mappings, or to change the default mapping with the "default" key.
//
// Note that this is not a concurrency safe map.
var ExtensionMapping = map[string]string{
	".css":    "text/css",
	".js":     "text/javascript",
	".html":   "text/html",
	".htm":    "text/html",
	".png":    "image/png",
	".jpeg":   "image/jpeg",
	".jpg":    "image/jpeg",
	"default": "application/octet-stream",
}

// ContentByExtension will match the filename extension to a MIME type using the current state of the [ExtensionMapping].
func ContentByExtension(filename string, data io.Reader) http.HandlerFunc {
	contentType, ok := ExtensionMapping[filepath.Ext(filename)]
	if !ok {
		return ContentHandler(ExtensionMapping["default"], data)
	}
	return ContentHandler(contentType, data)
}

// EmbeddedHandler will serve content from an [embed.FS], and try to resolve the content type using the file extension.
// A trim path may be specified, which will trim the prefix from the request path to construct a valid reference within the FS.
// An append prefix may also be added to allow using a different handler prefix than what would normally be expected to reference files in the FS.
func EmbeddedHandler(fs *embed.FS, trimPrefix string, appendPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		searchPath := r.URL.Path
		if len(trimPrefix) > 0 {
			searchPath = strings.TrimPrefix(searchPath, trimPrefix)
		}
		if len(appendPrefix) > 0 {
			searchPath = filepath.ToSlash(appendPrefix) + "/" + searchPath
		}
		searchPath = strings.TrimPrefix(searchPath, "/")
		f, err := fs.Open(searchPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(404)
				return
			}
			w.WriteHeader(500)
			return
		}
		defer func() {
			_ = f.Close()
		}()
		contentType, ok := ExtensionMapping[searchPath]
		if !ok {
			contentType = ExtensionMapping["default"]
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = io.Copy(w, f)
	}
}
