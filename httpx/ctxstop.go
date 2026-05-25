package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const (
	DefaultServerStopTimeout = 5 * time.Second // DefaultServerStopTimeout is the default timeout when waiting for a server to fully stop.
)

// ListenAndServeCtx will call [http.Server.ListenAndServe] and respond to context cancellation to shut down the server.
// An optional shutdownTimeout may be passed to override the default 5 second timeout.
func ListenAndServeCtx(ctx context.Context, srv *http.Server, shutdownTimeout ...time.Duration) error {
	return listenCtx(ctx, srv.ListenAndServe, srv.Shutdown, shutdownTimeout...)
}

// ListenAndServeTLSCtx will call [http.Server.ListenAndServeTLS] and respond to context cancellation to shut down the server.
// An optional shutdownTimeout may be passed to override the default 5 second timeout.
func ListenAndServeTLSCtx(ctx context.Context, srv *http.Server, certFile, keyFile string, shutdownTimeout ...time.Duration) error {
	return listenCtx(ctx, func() error {
		return srv.ListenAndServeTLS(certFile, keyFile)
	}, srv.Shutdown, shutdownTimeout...)
}

func listenCtx(ctx context.Context, serveFn func() error, shutdownFn func(context.Context) error, shutdownTimeout ...time.Duration) error {
	srvErrs := make(chan error)
	go func() {
		defer close(srvErrs)
		if err := serveFn(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				srvErrs <- err
			}
		}
	}()

	select {
	case err, more := <-srvErrs:
		if !more {
			return nil
		}
		return err
	case <-ctx.Done():
		var (
			timeout  context.Context
			cancel   context.CancelFunc
			stopTime = DefaultServerStopTimeout
		)
		if len(shutdownTimeout) > 0 {
			stopTime = shutdownTimeout[0]
		}
		timeout, cancel = context.WithTimeout(context.Background(), stopTime)
		defer cancel()
		if err := shutdownFn(timeout); err != nil {
			return err
		}
	}
	return nil
}
