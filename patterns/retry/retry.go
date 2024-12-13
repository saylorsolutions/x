package retry

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/x/contextx"
	"time"
)

// Iteration is a function that is called for each iteration of a loop, returning whether the error and whether it can be retried.
// When no error is returned, the loop will exit early with a nil error.
// Returning true with an error will attempt to retry the iteration.
// Returning false with an error will return early with the error.
// An error is returned when false is also returned from the Iteration, or the max retries has been reached.
type Iteration = func() (bool, error)

// Settings defines the backoff behavior for [Do].
type Settings struct {
	Context            context.Context
	TimeBetweenRetries time.Duration // This sets the initial delay between retries.
	BackoffFactor      float64       // This value multiplies TimeBetweenRetries between loop iterations, and should be >= 1.
	MaxTries           int           // This defines the maximum number of retries, and should be > 1.
}

func (s Settings) Copy() Settings {
	return Settings{
		Context:            s.Context,
		TimeBetweenRetries: s.TimeBetweenRetries,
		BackoffFactor:      s.BackoffFactor,
		MaxTries:           s.MaxTries,
	}
}

var (
	ErrInvalidSettings = errors.New("invalid settings")
	ErrMaxRetries      = errors.New("max tries exceeded")
)

type maxRetriesError struct {
	loopErr error
}

func (e *maxRetriesError) Error() string {
	return fmt.Sprintf("%v: %v", ErrMaxRetries, e.loopErr)
}

func (e *maxRetriesError) Unwrap() []error {
	return []error{ErrMaxRetries, e.loopErr}
}

// Do retries the given [Iteration] for a max of maxTries times.
// There is no delay between retries for this function.
func Do(maxTries int, iteration Iteration) error {
	return WithSettings(Settings{BackoffFactor: 1, MaxTries: maxTries}, iteration)
}

// WithSettings allows passing [Settings] to the retry loop to tune the operation.
func WithSettings(settings Settings, iteration Iteration) error {
	if settings.MaxTries <= 1 {
		return fmt.Errorf("%w: max tries should be > 1", ErrInvalidSettings)
	}
	if settings.BackoffFactor < 1 {
		return fmt.Errorf("%w: backoff factor should be >= 1", ErrInvalidSettings)
	}
	if settings.TimeBetweenRetries < 0 {
		return fmt.Errorf("%w: time between retries should be >= 0", ErrInvalidSettings)
	}
	var (
		shouldRetry bool
		iterErr     error
	)
	for i := 0; i < settings.MaxTries; i++ {
		// Delays and context checks
		if i > 0 && settings.TimeBetweenRetries > 0 {
			if settings.Context != nil {
				select {
				case <-settings.Context.Done():
					return settings.Context.Err()
				case <-time.After(settings.TimeBetweenRetries):
					// Timeout elapsed
				}
			} else {
				time.Sleep(settings.TimeBetweenRetries)
			}
			settings.TimeBetweenRetries = time.Duration(float64(settings.TimeBetweenRetries) * settings.BackoffFactor)
		} else if contextx.IsDone(settings.Context) {
			return settings.Context.Err()
		}

		// Try the loop
		shouldRetry, iterErr = iteration()
		if iterErr != nil {
			if shouldRetry {
				continue
			}
		}
		return iterErr
	}
	if iterErr != nil {
		return &maxRetriesError{iterErr}
	}
	return nil
}
