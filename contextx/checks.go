package contextx

import "context"

func IsDone(ctx context.Context) bool {
	if ctx == nil {
		// Returning false in this case so the caller doesn't attempt to extract the context error.
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
