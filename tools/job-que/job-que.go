package jobque

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// JobQue serializes function execution using a mutex.
type JobQue[T any] struct {
	mu sync.Mutex
}

func NewJobQue[T any]() *JobQue[T] {
	return &JobQue[T]{}
}

// Exec runs the given function in a serialized manner using the JobQue's mutex.
// This method is backward compatible and has no timeout.
func (q *JobQue[T]) Exec(fn func() (T, error)) (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return fn()
}

// ExecWithTimeout executes a function with a timeout and comprehensive logging.
func (q *JobQue[T]) ExecWithTimeout(timeout time.Duration, operationName string, contextInfo string, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return q.ExecWithContext(ctx, operationName, contextInfo, fn)
}

// ExecWithContext executes a function with context cancellation support and comprehensive logging.
func (q *JobQue[T]) ExecWithContext(ctx context.Context, operationName string, contextInfo string, fn func() (T, error)) (T, error) {
	start := time.Now()

	// Try to acquire lock with context
	lockAcquired := make(chan struct{})
	go func() {
		q.mu.Lock()
		close(lockAcquired)
	}()

	select {
	case <-ctx.Done():
		duration := time.Since(start)
		logTimeout(operationName, ctx, duration, contextInfo, "waiting for lock")
		var zero T
		return zero, fmt.Errorf("%s exceeded timeout after %v (waiting for lock)", operationName, duration)
	case <-lockAcquired:
		// Lock acquired, continue
	}
	defer q.mu.Unlock()

	lockWaitTime := time.Since(start)
	if lockWaitTime > time.Second {
		// Log if acquiring lock took > 1 second (potential contention)
		_, file, line, _ := runtime.Caller(1)
		log.Printf("[WARNING] Job queue lock contention: waited %v for lock - %s - %s:%d",
			lockWaitTime, contextInfo, file, line)
	}

	// Execute function with timeout
	resultCh := make(chan struct {
		result T
		err    error
	}, 1)

	go func() {
		result, err := fn()
		resultCh <- struct {
			result T
			err    error
		}{result, err}
	}()

	select {
	case <-ctx.Done():
		duration := time.Since(start)
		logTimeout(operationName, ctx, duration, contextInfo, "executing function")
		var zero T
		return zero, fmt.Errorf("%s exceeded timeout after %v", operationName, duration)
	case res := <-resultCh:
		return res.result, res.err
	}
}

func logTimeout(operationName string, ctx context.Context, duration time.Duration, contextInfo string, phase string) {
	_, file, line, _ := runtime.Caller(3)

	var timeoutDuration time.Duration
	if deadline, ok := ctx.Deadline(); ok {
		timeoutDuration = time.Until(deadline) + duration // Calculate original timeout
	}

	log.Printf("[TIMEOUT] %s exceeded timeout after %v (%s)\n"+
		"  Timeout: %v\n"+
		"  Actual Duration: %v\n"+
		"  Context: %s\n"+
		"  Caller: %s:%d",
		operationName,
		duration,
		phase,
		timeoutDuration,
		duration,
		contextInfo,
		file,
		line)
}
