package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"
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
		return
	}()

	select {
	case err, more := <-srvErrs:
		if !more {
			return nil
		}
		return err
	case <-ctx.Done():
		var (
			timeout context.Context
			cancel  context.CancelFunc
		)
		if len(shutdownTimeout) > 0 {
			timeout, cancel = context.WithTimeout(context.Background(), shutdownTimeout[0])
		} else {
			timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		}
		defer cancel()
		if err := shutdownFn(timeout); err != nil {
			return err
		}
	}
	return nil
}
