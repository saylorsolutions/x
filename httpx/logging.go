package httpx

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"
)

type loggingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (l *loggingWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.statusCode = statusCode
}

// RequestLogger is a type that can log HTTP requests received by a server.
type RequestLogger interface {
	Log(statusCode int, method, path string, duration time.Duration)
}

type RequestLoggerFunc func(statusCode int, method, path string, dur time.Duration)

func (f RequestLoggerFunc) Log(statusCode int, method, path string, dur time.Duration) {
	f(statusCode, method, path, dur)
}

// LoggingMiddleware will log each request to the given [http.Handler], including status code, method, path, and duration.
func LoggingMiddleware(logger RequestLogger, next http.Handler) http.Handler {
	if logger == nil {
		panic("nil logger")
	}
	if next == nil {
		panic("nil handler")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingWriter{w, http.StatusOK}
		start := time.Now()
		defer func() {
			dur := time.Since(start)
			code := lw.statusCode
			logger.Log(code, r.Method, r.URL.Path, dur)
		}()
		next.ServeHTTP(lw, r)
	})
}

// StdLogger returns a [RequestLogger] that wraps a [*log.Logger].
func StdLogger(l *log.Logger) RequestLogger {
	return RequestLoggerFunc(func(statusCode int, method, path string, dur time.Duration) {
		l.Println(statusCode, method, path, dur)
	})
}

// SlogLogger returns a [RequestLogger] that wraps a [*slog.Logger], and logs at the provided level.
func SlogLogger(l *slog.Logger, ctx context.Context, level slog.Level) RequestLogger {
	return RequestLoggerFunc(func(statusCode int, method, path string, dur time.Duration) {
		l.Log(ctx, level, "", "statusCode", statusCode, "method", method, "path", path, "duration", dur)
	})
}
