package httpx

import (
	"net/http"
)

// Middleware is a function that wraps another [http.Handler] to inject logic before or after the handler is run.
type Middleware func(next http.Handler) http.Handler

// Wrap will wrap the given [http.Handler], such that all given [Middleware] will be executed in the order provided.
// This can be used as a shorthand for applying many middleware layers while avoiding telescoping.
//
// If no middleware are provided, then the handler will be returned unchanged.
func Wrap(next http.Handler, middlewares ...Middleware) http.Handler {
	if next == nil {
		panic("nil handler")
	}
	// Wrapped in reverse order, so they're executed in parameter order.
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}
