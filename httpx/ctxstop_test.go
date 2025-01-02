package httpx

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestListenCtx(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		var (
			srv                            = make(chan struct{})
			serverListened, serverShutdown atomic.Bool
			serveFn                        = func() error {
				serverListened.Store(true)
				<-srv
				return nil
			}
			shutdownFn = func(ctx context.Context) error {
				serverShutdown.Store(true)
				close(srv)
				return nil
			}
		)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, listenCtx(ctx, serveFn, shutdownFn, 500*time.Millisecond))
		assert.True(t, serverListened.Load(), "Server listen function should have been called")
		assert.True(t, serverShutdown.Load(), "Server shutdown function should have been called")
	})
	t.Run("Server error propagated", func(t *testing.T) {
		var (
			srv                                  = make(chan struct{})
			errServerListened, errServerShutdown atomic.Bool
			errTestShutdown                      = errors.New("test error")
			errListenFn                          = func() error {
				defer close(srv)
				errServerListened.Store(true)
				return errTestShutdown
			}
			errShutdownFn = func(ctx context.Context) error {
				errServerShutdown.Store(true)
				return nil
			}
		)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.ErrorIs(t, listenCtx(ctx, errListenFn, errShutdownFn, 500*time.Millisecond), errTestShutdown)
		assert.True(t, errServerListened.Load(), "Server listen function should have been called")
		assert.False(t, errServerShutdown.Load(), "Server shutdown function should NOT have been called because the listener returns an error")
	})
}
