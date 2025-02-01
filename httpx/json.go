package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	HeaderContentType = "Content-Type"
)

var (
	ContentTypeJSON = "application/json" // This can be used to customize the content type reported to the client.
)

// JSONHandler is a function that accepts a JSON payload (specified with T), and returns a JSON response (specified with R).
type JSONHandler[T any, R any] func(body *T) (*R, error)

// JSONErrorHandler handles error conditions in [HandleJSON] to return a JSON representation of the error.
// This kind of function can be defined once and reused to establish a consistent policy.
//
// Within [HandleJSON], an error related to interpreting information from the client will be wrapped by [ErrClientError].
// Other errors will be wrapped by [ErrServerError].
// This allows the error handler to make specific decisions about how to report the issue.
// For example, this could be used to log server issues while just returning the response for client errors.
type JSONErrorHandler[E any] func(err error) E

// HandleJSON produces a [http.Handler] from a [JSONErrorHandler] and [JSONHandler] pair.
// It will handle deserialization of the JSON request payload, serialization of the JSON response payload, and serialization of JSON error responses.
// This will also handle closing the request body to ensure that resource usage is kept minimal.
// Client errors will result in a 400 status code being sent to the client. All other errors will result in a 500 status code.
func HandleJSON[T any, R any, E any](errHandler JSONErrorHandler[E], handler JSONHandler[T, R]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()
		var request T
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			errVal := errHandler(fmt.Errorf("%w: %v", ErrClientError, err))
			w.WriteHeader(400)
			w.Header().Set(HeaderContentType, ContentTypeJSON)
			_ = json.NewEncoder(w).Encode(errVal)
			return
		}
		resp, err := handler(&request)
		if err != nil {
			errVal := errHandler(fmt.Errorf("%w: %v", ErrServerError, err))
			w.WriteHeader(500)
			w.Header().Set(HeaderContentType, ContentTypeJSON)
			_ = json.NewEncoder(w).Encode(errVal)
			return
		}
		out, err := json.Marshal(resp)
		if err != nil {
			errVal := errHandler(fmt.Errorf("%w: %v", ErrServerError, err))
			w.WriteHeader(500)
			w.Header().Set(HeaderContentType, ContentTypeJSON)
			_ = json.NewEncoder(w).Encode(errVal)
			return
		}
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		_, err = w.Write(out)
		if err != nil {
			// Notify the handler in case the user wants to log this.
			// We've already handled any serialization issue, so this should largely be an unrecoverable I/O issue.
			_ = errHandler(fmt.Errorf("%w: %v", ErrServerError, err))
		}
	})
}
