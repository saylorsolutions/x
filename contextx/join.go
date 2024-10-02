package contextx

import (
	"context"
	"errors"
	"sync"
	"time"
)

type jointContext struct {
	a, b      context.Context
	done      chan struct{}
	doClose   func()
	doMonitor func()
	valuer    JoinValuer
}

func (c *jointContext) Done() <-chan struct{} {
	select {
	// In these cases, the done channel can be closed without monitoring.
	case <-c.a.Done():
		c.doClose()
	case <-c.b.Done():
		c.doClose()
	default:
		// The monitor goroutine is only started if needed.
		c.doMonitor()
	}
	// Since c.doClose and c.doMonitor are once functions, there's no risk of race conditions.
	return c.done
}

// Deadline returns the closes deadline reported from either [context.Context].
func (c *jointContext) Deadline() (time.Time, bool) {
	atime, aok := c.a.Deadline()
	btime, bok := c.b.Deadline()
	if !aok {
		return btime, bok
	}
	if !bok {
		return atime, aok
	}
	// Report the closest deadline
	if atime.Before(btime) {
		return atime, true
	}
	if btime.Before(atime) {
		return btime, true
	}
	return atime, aok
}

// Err uses [errors.Join] which will return both errors, the non-nil error, or nil.
func (c *jointContext) Err() error {
	return errors.Join(c.a.Err(), c.b.Err())
}

func (c *jointContext) Value(key any) any {
	aval, bval := c.a.Value(key), c.b.Value(key)
	if aval == nil {
		return bval
	}
	if bval == nil {
		return aval
	}
	if c.valuer != nil {
		return c.valuer.PickValue(aval, bval)
	}
	return aval
}

// Join will associate two or more [context.Context] together, such that cancellation, deadlines, and errors are reported together.
//
// If both input [context.Context] have a Deadline, then the one closest to the current time will be reported.
//
// When checking if the joined [context.Context] has been cancelled, if either is done at the time of check, then the joined context is cancelled.
// If neither has been cancelled, then a goroutine is created to monitor when either [context.Context] is cancelled, cancelling the joint context.
//
// Values are a bit more ambiguous.
// If multiple [context.Context] have a value for the same key, then there's no way for [Join] to pick the more correct value to return.
// In this case the first non-nil value is returned, and it might not be obvious to the caller which value that is.
// If you'd prefer to specify which value should be returned given multiple values, use [JoinWithValuer].
//
// If any [context.Context] is nil, then [Join] will panic.
func Join(a, b context.Context, others ...context.Context) context.Context {
	return JoinWithValuer(nil, a, b, others...)
}

// JoinValuer allows the caller to solve for ambiguity while picking a value in multiple [context.Context] with the same key.
type JoinValuer interface {
	PickValue(a, b any) any // PickValue will be called to determine which value should be returned from a joint context if multiple underlying [context.Context] have a value for the same key.
}

type JoinValuerFunc func(a, b any) any

func (f JoinValuerFunc) PickValue(a, b any) any {
	return f(a, b)
}

// JoinWithValuer is the same as [Join], but allows the caller to specify a [JoinValuer] to inject logic for picking the more correct context value in ambiguous cases.
//
// If any [context.Context] is nil, then [JoinWithValuer] will panic.
func JoinWithValuer(valuer JoinValuer, a, b context.Context, others ...context.Context) context.Context {
	if a == nil || b == nil {
		panic("nil context")
	}
	doneCh := make(chan struct{})
	closer := sync.OnceFunc(func() {
		close(doneCh)
	})
	joint := &jointContext{
		a:       a,
		b:       b,
		done:    doneCh,
		doClose: closer,
		valuer:  valuer,
	}
	monitor := func() {
		select {
		case <-joint.a.Done():
		case <-joint.b.Done():
		}
		joint.doClose()
	}
	joint.doMonitor = sync.OnceFunc(func() {
		go monitor()
	})
	if len(others) > 0 {
		return Join(joint, others[0], others[1:]...)
	}
	return joint
}
