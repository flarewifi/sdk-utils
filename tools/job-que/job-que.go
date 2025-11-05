package jobque

import "sync"

// Exec runs the given function after waiting for any
// previous call on the same mutex to finish.
//
// It uses generics so you can return any type (T) along with an error.
func Exec[T any](mu *sync.Mutex, fn func() (T, error)) (T, error) {
	mu.Lock()
	defer mu.Unlock()

	return fn()
}
