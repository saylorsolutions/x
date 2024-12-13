package retry

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	t.Run("Immediate failure", func(t *testing.T) {
		err := Do(3, testFailingIterator)
		assert.Error(t, err)
		assert.ErrorIs(t, err, testErrIntentional)
		assert.False(t, errors.Is(err, ErrMaxRetries), "Should not be a retry error")
	})
	t.Run("Retry failure", func(t *testing.T) {
		err := Do(3, testRetryableIterator)
		assert.Error(t, err)
		assert.ErrorIs(t, err, testErrIntentional)
		assert.True(t, errors.Is(err, ErrMaxRetries), "Should be a retry error")
		assert.Contains(t, err.Error(), ErrMaxRetries.Error())
		assert.Contains(t, err.Error(), testErrIntentional.Error())
	})
	t.Run("Successful attempt", func(t *testing.T) {
		err := Do(3, testPassingIterator)
		assert.NoError(t, err)
	})
}

func TestWithSettings(t *testing.T) {
	settings := Settings{
		Context:            nil,
		TimeBetweenRetries: 50 * time.Millisecond,
		BackoffFactor:      1.2,
		MaxTries:           3,
	}
	t.Run("Should not execute with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		settings := settings.Copy()
		settings.Context = ctx
		err := WithSettings(settings, func() (bool, error) {
			t.Error("Should not have called the Iterator")
			return false, nil
		})
		assert.ErrorIs(t, err, context.Canceled)
	})
	t.Run("Should take more than 110ms to loop", func(t *testing.T) {
		start := time.Now()
		err := WithSettings(settings, testRetryableIterator)
		dur := time.Since(start)
		assert.Greater(t, dur, 110*time.Millisecond)
		assert.ErrorIs(t, err, ErrMaxRetries)
	})
	t.Run("Invalid max tries", func(t *testing.T) {
		settings := settings.Copy()
		settings.MaxTries = 1
		err := WithSettings(settings, testPassingIterator)
		assert.ErrorIs(t, err, ErrInvalidSettings)
	})
	t.Run("Invalid time between retries", func(t *testing.T) {
		settings := settings.Copy()
		settings.TimeBetweenRetries = -1
		err := WithSettings(settings, testPassingIterator)
		assert.ErrorIs(t, err, ErrInvalidSettings)
	})
	t.Run("Invalid backoff factor", func(t *testing.T) {
		settings := settings.Copy()
		settings.BackoffFactor = 0.5
		err := WithSettings(settings, testPassingIterator)
		assert.ErrorIs(t, err, ErrInvalidSettings)
	})
}

var testErrIntentional = errors.New("intentional error")

func testFailingIterator() (bool, error) {
	return false, testErrIntentional
}

func testRetryableIterator() (bool, error) {
	return true, testErrIntentional
}

func testPassingIterator() (bool, error) {
	return false, nil
}
