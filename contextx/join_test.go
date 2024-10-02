package contextx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestJointContext_Done(t *testing.T) {
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	aCancelled := Join(ctx, bg)
	cancel()
	assert.True(t, IsDone(aCancelled), "Joint context should have been cancelled")
	assert.Error(t, aCancelled.Err())

	ctx, cancel = context.WithCancel(bg)
	bCancelled := Join(bg, ctx)
	cancel()
	assert.True(t, IsDone(bCancelled), "Joint context should have been cancelled")
	assert.Error(t, bCancelled.Err())

	ctx, cancel = context.WithCancel(bg)
	asyncCancelled := Join(bg, ctx)
	assert.False(t, IsDone(asyncCancelled), "Joint context should NOT have been cancelled")
	assert.NoError(t, asyncCancelled.Err())
	cancel()
	assert.True(t, IsDone(asyncCancelled), "Joint context should have been cancelled")
	assert.Error(t, ctx.Err())

	ctx, cancel = context.WithCancel(bg)
	awaitCancelled := Join(bg, ctx)
	doneCh := awaitCancelled.Done()
	assert.False(t, IsDone(awaitCancelled), "Joint context should NOT have been cancelled")
	assert.NoError(t, awaitCancelled.Err())
	cancel()
	select {
	case <-time.After(time.Second):
		t.Error("Should have closed from the monitor")
	case <-doneCh:
		t.Log("Monitor closed the channel")
	}
	assert.True(t, IsDone(awaitCancelled), "Joint context should have been cancelled")
	assert.Error(t, ctx.Err())
	assert.ErrorIs(t, awaitCancelled.Err(), context.Canceled)
}

func TestJointContext_Value(t *testing.T) {
	bg := context.Background()
	ctx := context.WithValue(bg, "key", "value")
	aValue := Join(ctx, bg)
	assert.Equal(t, "value", aValue.Value("key"))

	bValue := Join(bg, ctx)
	assert.Equal(t, "value", bValue.Value("key"))
}

func TestJoin(t *testing.T) {
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	joint := Join(ctx, bg, bg, bg)
	assert.False(t, IsDone(joint), "Should not be done yet")
	cancel()
	assert.True(t, IsDone(joint), "Should be done now")
}

func TestJointContext_Deadline(t *testing.T) {
	bg := context.Background()
	noDeadline := Join(bg, bg)
	deadline, ok := noDeadline.Deadline()
	assert.False(t, ok, "Should be no deadline reported")

	ctx, cancel := context.WithTimeout(bg, 100*time.Millisecond)
	defer cancel()
	joint := Join(ctx, bg)
	deadline, ok = joint.Deadline()
	assert.True(t, ok, "Should have returned a deadline")
	assert.True(t, deadline.After(time.Now()), "Deadline should not have elapsed yet")

	timeout, cancel2 := context.WithDeadline(bg, time.Now().Add(time.Second))
	defer cancel2()
	joint2 := Join(timeout, joint)
	deadline, ok = joint2.Deadline()
	assert.True(t, ok, "Should have returned a deadline")
	assert.True(t, deadline.After(time.Now()), "Deadline should not have elapsed yet")
}

func TestJoinWithValuer(t *testing.T) {
	bg := context.Background()
	actx := context.WithValue(bg, "key", 5)
	bctx := context.WithValue(bg, "key", 10)
	noValuer := Join(actx, bctx)
	val := noValuer.Value("key").(int)
	assert.True(t, val == 5 || val == 10, "Value should be one of the context values")

	withValuer := JoinValuerFunc(func(a, b any) any {
		aval, ok := a.(int)
		if !ok {
			return b
		}
		bval, ok := b.(int)
		if !ok {
			return a
		}
		if aval < bval {
			return aval
		}
		return bval
	})
	ctx := JoinWithValuer(withValuer, actx, bctx)
	assert.Equal(t, 5, ctx.Value("key"))
}
