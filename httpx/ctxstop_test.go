package httpx

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestListenCtx(t *testing.T) {
	var (
		srv                                                                  = make(chan bool)
		serverListened, serverShutdown, errServerListened, errServerShutdown bool
		listenFn                                                             = func() error {
			serverListened = true
			<-srv
			return nil
		}
		shutdownFn = func(ctx context.Context) error {
			serverShutdown = true
			close(srv)
			return nil
		}
		errTestShutdown = errors.New("test error")
		errListenFn     = func() error {
			errServerListened = true
			return errTestShutdown
		}
		errShutdownFn = func(ctx context.Context) error {
			errServerShutdown = true
			return nil
		}
	)

	t.Run("Happy path", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		time.AfterFunc(time.Second, func() {
			cancel()
		})
		assert.NoError(t, listenCtx(ctx, listenFn, shutdownFn, 500*time.Millisecond))
		assert.True(t, serverListened, "Server listen function should have been called")
		assert.True(t, serverShutdown, "Server shutdown function should have been called")
	})
	t.Run("Server error propagated", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		time.AfterFunc(time.Second, func() {
			cancel()
		})
		assert.ErrorIs(t, listenCtx(ctx, errListenFn, errShutdownFn, 500*time.Millisecond), errTestShutdown)
		assert.True(t, errServerListened, "Server listen function should have been called")
		assert.False(t, errServerShutdown, "Server shutdown function should NOT have been called because the listener returns an error")
	})
}
