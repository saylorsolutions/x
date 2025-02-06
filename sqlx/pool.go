package sqlx

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/env"
	"github.com/saylorsolutions/x/iterx"
	"github.com/saylorsolutions/x/syncx"
	"io"
	"log"
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
	keepAliveInterval time.Duration
	debugLogging      bool
}

// ConnectionFactory is a function that produces a new [Connection] on demand.
type ConnectionFactory[T Connection] func() (T, error)

// KeepAlive is a function that can be used to check the open status of an existing [Connection].
// This is usually implemented by pinging over the connection or executing a small query.
type KeepAlive[T Connection] func(T) error

type Pool[T Connection] struct {
	conf           poolConf
	doneMonitoring sync.WaitGroup
	factory        ConnectionFactory[T]
	keepAlive      KeepAlive[T]

	mux   sync.RWMutex
	conns []*poolConn[T]
}

func (p *Pool[T]) debug(args ...any) {
	if !p.conf.debugLogging {
		return
	}
	log.Println(append([]any{"[SQLX_POOLDEBUG]"}, args...)...)
}

func confErrf(msg string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: "+msg, ErrConfig)
	}
	return fmt.Errorf("%w: "+msg, append([]any{ErrConfig}, args...)...)
}

type PoolConfigOpt func(conf *poolConf) error

// OptMinConnections specifies a number of connections that should be retained, even when the [Pool] is idle.
// Maintaining some active connections can ensure that infrequent querying that would normally exceed the idle timeout can still be performant.
// The default is 0 to minimize resource consumption.
func OptMinConnections(min int) PoolConfigOpt {
	return func(conf *poolConf) error {
		if min < 0 {
			return confErrf("invalid minimum connection size %d", min)
		}
		conf.minConns = min
		return nil
	}
}

// OptAcquireTimeout specifies the maximum amount of time that the [Pool] should wait to acquire new connections.
// The default is 5 seconds.
func OptAcquireTimeout(max time.Duration) PoolConfigOpt {
	return func(conf *poolConf) error {
		if max <= 0 {
			return confErrf("invalid acquire timeout '%s'", max)
		}
		conf.acquireTimeout = max
		return nil
	}
}

// OptIdleBehavior specifies the bounds for considering a connection to be idle in the [Pool].
//   - idleTimeout is the amount of time that a connection must be available in the pool before being eligible for closure.
//   - idleCheckInterval is the interval that the pool will use to scan for idle connections.
//   - When a previously leased connection is released back to the pool, its idleTimeout is reset.
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

// OptKeepAliveInterval specifies the interval to run keep alive checks on available connections in the [Pool].
// Increasing this value can reduce load on the database from background queries, and reduce contention on the connections in the pool.
// Reducing this value can reduce the possibility that connections will be closed when acquired, requiring a new connection to be created.
func OptKeepAliveInterval(keepAliveInterval time.Duration) PoolConfigOpt {
	return func(conf *poolConf) error {
		if keepAliveInterval <= 0 {
			return confErrf("invalid keep alive interval '%s'", keepAliveInterval)
		}
		conf.keepAliveInterval = keepAliveInterval
		return nil
	}
}

// OptEnableDebugLogging enables internal logging for this [Pool].
//
// This may also be controlled by setting the environment variable SQLX_POOLDEBUG to a boolean value.
func OptEnableDebugLogging() PoolConfigOpt {
	return func(conf *poolConf) error {
		conf.debugLogging = true
		return nil
	}
}

func NewConnectionPool[T Connection](ctx context.Context, factory ConnectionFactory[T], keepAlive KeepAlive[T], maxPoolDepth int, opts ...PoolConfigOpt) (*Pool[T], error) {
	if ctx == nil {
		return nil, confErrf("context is required")
	}
	if factory == nil {
		return nil, confErrf("factory function is required")
	}
	if keepAlive == nil {
		return nil, confErrf("keepAlive function is required")
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
		keepAliveInterval: 3 * time.Second,
		debugLogging:      env.Bool("SQLX_POOLDEBUG", false),
	}
	for _, opt := range opts {
		if err := opt(conf); err != nil {
			return nil, err
		}
	}
	if conf.minConns > conf.maxConns {
		return nil, confErrf("minimum connections %d cannot be greater than max connections %d", conf.minConns, conf.maxConns)
	}
	pool := &Pool[T]{
		conf:      *conf,
		conns:     make([]*poolConn[T], conf.maxConns),
		factory:   factory,
		keepAlive: keepAlive,
	}
	pool.doneMonitoring.Add(2)
	go pool.idleMonitor()
	go pool.keepAliveLoop()
	for i := 0; i < conf.minConns; i++ {
		conn, err := factory()
		if err != nil {
			pool.debug("Failed to create connection for minimum pool size:", err)
			closeErr := pool.Close()
			return nil, errors.Join(err, closeErr)
		}
		pool.Release(conn)
	}
	return pool, nil
}

func (p *Pool[T]) idleMonitor() {
	const debugLabel = "[idleMonitor]"
	defer p.doneMonitoring.Done()
	ticker := time.NewTicker(p.conf.idleCheckInterval)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-p.conf.ctx.Done():
			p.debug(debugLabel, "context cancelled, exiting")
			return
		case now := <-ticker.C:
			syncx.LockFunc(&p.mux, func() {
				idle := p.available().FilterValues(func(conn *poolConn[T]) bool {
					return conn.idleDeadline.Before(now)
				})
				toExpire := idle.Count()
				if toExpire == 0 {
					p.debug(debugLabel, "no connections eligible for expiration")
					return
				}
				p.debug(debugLabel, "connections eligible for expiration:", toExpire)
				remaining := p.poolDepth() - toExpire
				if remaining < p.conf.minConns {
					// Need to limit the number of connections expired to maintain the minimum pool depth.
					limit := toExpire - (p.conf.minConns - remaining)
					p.debug(debugLabel, "limiting connection expiration to maintain minimum:", limit)
					idle = idle.Limit(limit)
				}
				idle.ForEach(func(i int, val *poolConn[T]) bool {
					if err := val.conn.Close(); err != nil {
						p.debug(debugLabel, "Failed to expire connection:", err)
					}
					p.conns[i] = nil
					p.debug(debugLabel, "one connection expired")
					return true
				})
			})
		}
	}
}

func (p *Pool[T]) keepAliveLoop() {
	const debugLabel = "[keepAliveLoop]"
	defer p.doneMonitoring.Done()
	ticker := time.NewTicker(p.conf.idleCheckInterval)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-p.conf.ctx.Done():
			p.debug(debugLabel, "context cancelled, exiting")
			return
		case <-ticker.C:
			syncx.LockFunc(&p.mux, func() {
				var (
					toRecycle    []int
					numAvailable = p.available().Count()
				)
				p.available().ForEach(func(idx int, slot *poolConn[T]) bool {
					if err := p.keepAlive(slot.conn); err != nil {
						p.debug(debugLabel, "connection returned error for keepAlive check:", err)
						// Attempt close just in case the connection is in a weird state.
						if err := slot.conn.Close(); err != nil {
							p.debug(debugLabel, "unable to close failing connection:", err)
						}
						toRecycle = append(toRecycle, idx)
					}
					return true
				})
				for _, idx := range toRecycle {
					p.debug(debugLabel, "one connection cleared for inactivity")
					p.conns[idx] = nil
				}
				if numAvailable-len(toRecycle) < p.conf.minConns {
					// Recreate connections to get back up to the minimum.
					toRecreate := p.conf.minConns - (numAvailable - len(toRecycle))
					p.debug(debugLabel, "need to recreate connections to maintain minimum:", toRecreate)
					for i := 0; i < toRecreate; i++ {
						newConn, err := p.acquireNew()
						if err != nil {
							p.debug(debugLabel, "unable to recreate connection from factory:", err)
							break
						}
						p.debug(debugLabel, "recreated one connection")
						p.release(newConn)
					}
				}
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
		p.debug("context cancelled, stopping acquisition")
		return mt, err
	}
	conn, found := func() (T, bool) {
		return p.acquireExisting()
	}()
	if found {
		p.debug("returning available connection")
		return conn, nil
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if err := p.conf.ctx.Err(); err != nil {
		p.debug("context cancelled, stopping acquisition")
		return mt, err
	}
	p.debug("creating new connection for acquisition")
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
	p.debug("pool exhausted")
	return mt, ErrPoolExhausted
}

func (p *Pool[T]) acquireExisting() (T, bool) {
	p.mux.Lock()
	defer p.mux.Unlock()
	var mt T

	conn, ok := p.available().FilterValues(func(slot *poolConn[T]) bool {
		err := p.keepAlive(slot.conn)
		if err != nil {
			p.debug("existing connection is not serviceable:", err)
			return false
		}
		return true
	}).Values().First()
	if ok {
		conn.state = stateLeased
		p.debug("existing connection available for leasing")
		return conn.conn, true
	}
	p.debug("no existing connections available")
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
		conn, err := p.factory()
		p.debug("[acquireGoroutine]", "finished acquiring new connection from factory")
		result.ResolveErr(conn, err)
	}()

	resultCh := syncx.FutureErrChannel(result)
	select {
	case errResult := <-resultCh:
		if errResult.Err != nil {
			p.debug("error acquiring new connection")
			return mt, fmt.Errorf("%w: %v", ErrFailedAcquire, errResult.Err)
		}
		p.debug("acquired new connection")
		return errResult.Result, nil
	case <-timeout.Done():
		go func() {
			const debugLabel = "[drainGoroutine]"
			// Immediately close if the factory function returns.
			r := <-resultCh
			p.debug(debugLabel, "factory function returned")
			if r.Err != nil {
				p.debug(debugLabel, "error returned from factory function:", r.Err)
				return
			}
			if r.Result != mt {
				if err := r.Result.Close(); err != nil {
					p.debug(debugLabel, "failed to close late connection:", err)
				}
			}
		}()
		p.debug("timed out waiting for factory function")
		return mt, fmt.Errorf("%w: timeout exceeded: %v", ErrFailedAcquire, timeout.Err())
	}
}

func (p *Pool[T]) Release(conn T) {
	var mt T
	if conn == mt {
		return
	}
	if err := p.conf.ctx.Err(); err != nil {
		p.debug("context cancelled, closing connection to be released")
		_ = conn.Close()
		return
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	p.release(conn)
}

func (p *Pool[T]) release(conn T) {
	if err := p.conf.ctx.Err(); err != nil {
		p.debug("context closed, closing connection and cancelling release")
		_ = conn.Close()
		return
	}

	idleDeadline := time.Now().Add(p.conf.idleTimeout)
	idx, val, ok := p.connections().FilterValues(func(slot *poolConn[T]) bool {
		return slot != nil && slot.state == stateLeased && slot.conn == conn
	}).First()
	if ok {
		if err := p.keepAlive(conn); err != nil {
			p.debug("released connection is not serviceable:", err)
			p.conns[idx] = nil
			return
		}
		p.debug("returning connection to existing slot")
		val.state = stateAvailable
		val.idleDeadline = idleDeadline
		return
	}
	if err := p.keepAlive(conn); err != nil {
		p.debug("released connection is not serviceable:", err)
		return
	}

	idx, ok = p.connections().FilterValues(func(slot *poolConn[T]) bool {
		return slot == nil
	}).Keys().First()
	if !ok {
		p.debug("no space available to release connection, closing")
		// No more space available, proactively close the connection.
		_ = conn.Close()
		return
	}
	p.conns[idx] = &poolConn[T]{
		conn:         conn,
		state:        stateAvailable,
		idleDeadline: idleDeadline,
	}
}

type PoolStats struct {
	FreeSlots            int
	LeasedConnections    int
	AvailableConnections int
	Utilization          float64
}

func (p *Pool[T]) Stats() PoolStats {
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
	p.debug("returning pool stats")
	return stats
}

func (p *Pool[T]) Close() error {
	p.debug("Pool.Close called")
	p.conf.cancel()
	p.doneMonitoring.Wait()
	p.mux.Lock()
	defer p.mux.Unlock()
	p.debug("closing existing connections")
	errs := make([]error, len(p.conns))
	for i, conn := range p.conns {
		if conn == nil {
			continue
		}
		errs[i] = conn.conn.Close()
	}
	p.debug("done closing existing connections")
	return errors.Join(errs...)
}
