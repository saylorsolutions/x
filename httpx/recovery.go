package httpx

import "net/http"

type PanicHandler interface {
	Handle(r any)
}

func RecoveryMiddleware(handler PanicHandler) Middleware {
	if handler == nil {
		panic("nil panic handler")
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("nil handler")
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					handler.Handle(r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
