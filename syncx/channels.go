package syncx

// Merge will create a goroutine to merge two or more channels into one unbuffered channel.
// Any T dispatched to any source channel will be output to the merged channel.
//
// Note that, given N source channels, N-1 goroutines will be created.
func Merge[T any](a, b <-chan T, others ...<-chan T) <-chan T {
	newCh := make(chan T)
	go func() {
		defer close(newCh)
		for {
			var dispatch T
			select {
			case val, more := <-a:
				if !more {
					a = nil
					continue
				}
				dispatch = val
			case val, more := <-b:
				if !more {
					b = nil
					continue
				}
				dispatch = val
			}
			newCh <- dispatch
		}
	}()
	if len(others) > 0 {
		a, b := newCh, others[0]
		return Merge(a, b, others[1:]...)
	}
	return newCh
}
