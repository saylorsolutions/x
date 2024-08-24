package syncx

import "sync"

func LockFunc(mux sync.Locker, fn func()) {
	mux.Lock()
	defer mux.Unlock()
	fn()
}

func LockFuncT[T any](mux sync.Locker, fn func() T) T {
	mux.Lock()
	defer mux.Unlock()
	return fn()
}

func LockFuncTErr[T any](mux sync.Locker, fn func() (T, error)) (T, error) {
	mux.Lock()
	defer mux.Unlock()
	return fn()
}

type RLocker interface {
	RLock()
	RUnlock()
}

func RLockFunc(mux RLocker, fn func()) {
	mux.RLock()
	defer mux.RUnlock()
	fn()
}

func RLockFuncT[T any](mux RLocker, fn func() T) T {
	mux.RLock()
	defer mux.RUnlock()
	return fn()
}

func RLockFuncTErr[T any](mux RLocker, fn func() (T, error)) (T, error) {
	mux.RLock()
	defer mux.RUnlock()
	return fn()
}
