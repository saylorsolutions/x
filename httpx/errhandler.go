package httpx

import (
	"errors"
	"net/http"
)

var (
	ErrClientError    = errors.New("client error")
	ErrServerError    = errors.New("server error")
	ErrAuthentication = errors.New("authentication error")
	ErrAuthorization  = errors.New("authorization error")
)

// ErrHandlerFunc is a function much like a [http.HandlerFunc], except that it returns an error.
// This can be used to more intuitively handle error conditions in HTTP handlers.
type ErrHandlerFunc = func(w http.ResponseWriter, r *http.Request) error

// ErrHandler adapts a handler function that returns an error to a standard [http.HandlerFunc].
// In the event of a non-nil error being returned, the type of the error will dictate the response, defaulting to 500 for unknown error types.
// The errors checked are: [ErrServerError], [ErrClientError], [ErrAuthentication], and [ErrAuthorization].
// To customize error handling behavior, use [ErrPolicy].
func ErrHandler(handler ErrHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			switch {
			case errors.Is(err, ErrClientError):
				w.WriteHeader(http.StatusBadRequest)
			case errors.Is(err, ErrAuthentication):
				w.WriteHeader(http.StatusUnauthorized)
			case errors.Is(err, ErrAuthorization):
				w.WriteHeader(http.StatusForbidden)
			case errors.Is(err, ErrServerError):
				fallthrough
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

// ErrPolicy creates a function that accepts an [ErrHandlerFunc], and runs errHandler with the returned error when it's non-nil.
// This can be used to wrap one or more ErrHandlerFunc with consistent, user defined error handling logic.
func ErrPolicy(errHandler func(w http.ResponseWriter, r *http.Request, err error)) func(ErrHandlerFunc) http.HandlerFunc {
	return func(handlerFunc ErrHandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := handlerFunc(w, r); err != nil {
				errHandler(w, r, err)
			}
		}
	}
}
