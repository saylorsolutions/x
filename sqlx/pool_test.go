package sqlx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

type mockConn struct {
	closed atomic.Bool
}

func (m *mockConn) Close() error {
	m.closed.Store(true)
	return nil
}

func newMockConn() (*mockConn, error) {
	return new(mockConn), nil
}

func TestConnPool_Acquire_Exhausted(t *testing.T) {
	pool, err := NewConnectionPool[*mockConn](context.TODO(), newMockConn, 1)
	require.NoError(t, err)
	require.NotNil(t, pool)

	first, err := pool.Acquire()
	assert.NoError(t, err)
	assert.NotNil(t, first)
	assert.Equal(t, 1.0, pool.PoolStats().Utilization)

	second, err := pool.Acquire()
	assert.ErrorIs(t, err, ErrPoolExhausted)
	assert.Nil(t, second)

	assert.NoError(t, pool.Close())
	assert.True(t, first.closed.Load())
}

func TestConnPool_Return(t *testing.T) {
	pool, err := NewConnectionPool[*mockConn](context.TODO(), newMockConn, 1,
		OptIdleBehavior(100*time.Millisecond, 75*time.Millisecond),
		OptMinConnections(0),
	)
	require.NoError(t, err)
	require.NotNil(t, pool)

	conn, err := pool.Acquire()
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, stateLeased, pool.conns[0].state)
	assert.Equal(t, 1.0, pool.PoolStats().Utilization)
	pool.Release(conn)
	assert.Equal(t, 0.0, pool.PoolStats().Utilization)
	assert.Equal(t, stateAvailable, pool.conns[0].state)
	assert.True(t, pool.conns[0].idleDeadline.After(time.Now()))
	time.Sleep(300 * time.Millisecond)
	assert.True(t, conn.closed.Load())
	assert.NoError(t, pool.Close())
}
