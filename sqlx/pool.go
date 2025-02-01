package sqlx

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/iterx"
	"github.com/saylorsolutions/x/syncx"
	"io"
	"sync"
	"time"
)

var (
	ErrFailedAcquire = errors.New("failed to acquire connection")
	ErrPoolExhausted = errors.New("pool is exhausted and cannot create new connections")
	ErrConfig        = errors.New("invalid configuration")
)

type connState int32

const (
	stateNone connState = iota
	stateAvailable
	stateLeased
)

// Connection describes a type that can be managed by the [Pool].
type Connection interface {
	io.Closer
	comparable
}

type poolConn[T Connection] struct {
	conn         T
	state        connState
	idleDeadline time.Time
}

type poolConf struct {
	ctx               context.Context
	cancel            context.CancelFunc
	minConns          int
	maxConns          int
	acquireTimeout    time.Duration
	idleTimeout       time.Duration
	idleCheckInterval time.Duration
}

type Pool[T Connection] struct {
	conf           poolConf
	doneMonitoring sync.WaitGroup
	factory        func() (T, error)

	mux   sync.RWMutex
	conns []*poolConn[T]
}

func confErrf(msg string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: "+msg, ErrConfig)
	}
	return fmt.Errorf("%w: "+msg, append([]any{ErrConfig}, args...)...)
}

type PoolConfigOpt func(conf *poolConf) error

func OptMinConnections(min int) PoolConfigOpt {
	return func(conf *poolConf) error {
		if min < 0 {
			return confErrf("invalid minimum connection size %d", min)
		}
		conf.minConns = min
		return nil
	}
}

func OptAcquireTimeout(max time.Duration) PoolConfigOpt {
	return func(conf *poolConf) error {
		if max <= 0 {
			return confErrf("invalid acquire timeout '%s'", max)
		}
		conf.acquireTimeout = max
		return nil
	}
}

func OptIdleBehavior(idleTimeout, idleCheckInterval time.Duration) PoolConfigOpt {
	return func(conf *poolConf) error {
		if idleTimeout <= 0 {
			return confErrf("invalid idle timeout duration '%s'", idleTimeout)
		}
		if idleCheckInterval <= 0 {
			return confErrf("invalid idle check interval '%s'", idleCheckInterval)
		}
		conf.idleTimeout = idleTimeout
		conf.idleCheckInterval = idleCheckInterval
		return nil
	}
}

func NewConnectionPool[T Connection](ctx context.Context, factory func() (T, error), maxPoolDepth int, opts ...PoolConfigOpt) (*Pool[T], error) {
	if ctx == nil {
		return nil, confErrf("context is required")
	}
	if factory == nil {
		return nil, confErrf("factory function is required")
	}
	if maxPoolDepth <= 0 {
		return nil, confErrf("max connections (%d) must be greater than zero", maxPoolDepth)
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defaultIdle := 2 * time.Minute
	conf := &poolConf{
		ctx:               ctx,
		cancel:            cancel,
		maxConns:          maxPoolDepth,
		acquireTimeout:    5 * time.Second,
		idleTimeout:       defaultIdle,
		idleCheckInterval: defaultIdle,
	}
	for _, opt := range opts {
		if err := opt(conf); err != nil {
			return nil, err
		}
	}
	pool := &Pool[T]{
		conf:    *conf,
		conns:   make([]*poolConn[T], conf.maxConns),
		factory: factory,
	}
	pool.doneMonitoring.Add(1)
	go pool.idleMonitor()
	return pool, nil
}

func (p *Pool[T]) idleMonitor() {
	defer p.doneMonitoring.Done()
	ticker := time.NewTicker(p.conf.idleCheckInterval)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-p.conf.ctx.Done():
			return
		case now := <-ticker.C:
			syncx.LockFunc(&p.mux, func() {
				idle := p.available().FilterValues(func(conn *poolConn[T]) bool {
					return conn.idleDeadline.Before(now)
				})
				toExpire := idle.Count()
				if toExpire == 0 {
					return
				}
				remaining := p.poolDepth() - toExpire
				if remaining < p.conf.minConns {
					// Need to limit the number of connections expired to maintain the minimum pool depth.
					idle = idle.Limit(toExpire - (p.conf.minConns - remaining))
				}
				idle.ForEach(func(i int, val *poolConn[T]) bool {
					_ = val.conn.Close()
					p.conns[i] = nil
					return true
				})
			})
		}
	}
}

func (p *Pool[T]) connections() iterx.MapIter[int, *poolConn[T]] {
	return iterx.SliceMap(p.conns)
}

func (p *Pool[T]) available() iterx.MapIter[int, *poolConn[T]] {
	return p.connections().FilterValues(func(conn *poolConn[T]) bool {
		return conn != nil && conn.state == stateAvailable
	})
}

func (p *Pool[T]) poolDepth() int {
	return iterx.Select(p.conns, func(val *poolConn[T]) bool {
		return val != nil
	}).Count()
}

func (p *Pool[T]) Acquire() (T, error) {
	var mt T
	if err := p.conf.ctx.Err(); err != nil {
		return mt, err
	}
	conn, found := func() (T, bool) {
		return p.acquireExisting()
	}()
	if found {
		return conn, nil
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if err := p.conf.ctx.Err(); err != nil {
		return mt, err
	}
	if p.poolDepth() < p.conf.maxConns {
		newConn, err := p.acquireNew()
		if err != nil {
			return mt, err
		}
		for i, element := range p.conns {
			if element == nil {
				p.conns[i] = &poolConn[T]{
					conn:  newConn,
					state: stateLeased,
				}
				return newConn, nil
			}
		}
	}
	return mt, ErrPoolExhausted
}

func (p *Pool[T]) acquireExisting() (T, bool) {
	p.mux.Lock()
	defer p.mux.Unlock()
	var mt T

	conn, ok := p.available().Values().First()
	if ok {
		conn.state = stateLeased
		return conn.conn, true
	}
	return mt, false
}

func (p *Pool[T]) acquireNew() (T, error) {
	var (
		timeout, cancel = context.WithTimeout(p.conf.ctx, p.conf.acquireTimeout)
		result          = syncx.NewFutureErr[T]()
		mt              T
	)
	defer cancel()
	go func() {
		result.ResolveErr(p.factory())
	}()

	resultCh := syncx.FutureErrChannel(result)
	select {
	case errResult := <-resultCh:
		if errResult.Err != nil {
			return mt, fmt.Errorf("%w: %v", ErrFailedAcquire, errResult.Err)
		}
		return errResult.Result, nil
	case <-timeout.Done():
		go func() {
			// Immediately close if the factory function returns.
			r := <-resultCh
			if r.Err != nil {
				return
			}
			if r.Result != mt {
				_ = r.Result.Close()
			}
		}()
		return mt, fmt.Errorf("%w: timeout exceeded: %v", ErrFailedAcquire, timeout.Err())
	}
}

func (p *Pool[T]) Release(conn T) {
	var mt T
	if conn == mt {
		return
	}
	if err := p.conf.ctx.Err(); err != nil {
		_ = conn.Close()
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if err := p.conf.ctx.Err(); err != nil {
		_ = conn.Close()
		return
	}

	now := time.Now()
	val, ok := p.connections().FilterValues(func(val *poolConn[T]) bool {
		return val != nil && val.state == stateLeased && val.conn == conn
	}).Values().First()
	if ok {
		val.state = stateAvailable
		val.idleDeadline = now.Add(p.conf.idleTimeout)
		return
	}

	for i, space := range p.conns {
		if space == nil {
			p.conns[i] = &poolConn[T]{
				conn:         conn,
				state:        stateAvailable,
				idleDeadline: now.Add(p.conf.idleTimeout),
			}
			return
		}
	}
	// No more space available, proactively close the connection.
	_ = conn.Close()
}

type PoolStats struct {
	FreeSlots            int
	LeasedConnections    int
	AvailableConnections int
	Utilization          float64
}

func (p *Pool[T]) PoolStats() PoolStats {
	p.mux.RLock()
	defer p.mux.RUnlock()
	segments := iterx.PartitionSlice(p.connections().Values(), func(conn *poolConn[T]) connState {
		if conn == nil {
			return stateNone
		}
		return conn.state
	})
	stats := PoolStats{
		FreeSlots:            segments[stateNone].Count(),
		LeasedConnections:    segments[stateLeased].Count(),
		AvailableConnections: segments[stateAvailable].Count(),
	}
	stats.Utilization = float64(stats.LeasedConnections) / float64(p.conf.maxConns)
	return stats
}

func (p *Pool[T]) Close() error {
	p.conf.cancel()
	p.doneMonitoring.Wait()
	p.mux.Lock()
	defer p.mux.Unlock()
	errs := make([]error, len(p.conns))
	for i, conn := range p.conns {
		if conn == nil {
			continue
		}
		errs[i] = conn.conn.Close()
	}
	return errors.Join(errs...)
}
