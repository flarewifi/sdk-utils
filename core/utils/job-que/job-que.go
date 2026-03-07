// Package jobque provides a generic single-worker job queue for serializing
// function execution. Jobs are executed one at a time in FIFO order by a
// single background worker goroutine.
//
// Key features:
//   - Generic return values via Go generics
//   - Context and timeout support for cancellation
//   - Automatic panic recovery at both job and worker level
//   - Contention logging when jobs wait too long in queue
//   - Caller file/line tracking for diagnostics
//
// Context Cancellation Behavior:
//
// Context cancellation (or timeout expiration) does NOT prevent execution of
// an already-enqueued job. Once a job enters the queue, the worker will
// eventually execute it. If the caller's context expires before execution
// completes, the caller receives a context error, but the job may still run
// and produce side effects. This is an inherent limitation of buffered channel
// queues — jobs cannot be removed once enqueued.
//
// Callers should design submitted functions to be:
//   - Idempotent, or
//   - Safe to execute even after the caller has given up waiting
package jobque

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrQueueClosed is returned by Exec/ExecWithContext/ExecWithTimeout when the
// queue has already been closed via Close().
var ErrQueueClosed = errors.New("job queue is closed")

type result[T any] struct {
	val T
	err error
}

type job[T any] struct {
	ctx           context.Context
	operationName string
	contextInfo   string
	fn            func() (T, error)
	resp          chan result[T]
	enqueuedAt    time.Time
	callerFile    string
	callerLine    int
}

type JobQueue[T any] struct {
	jobs                chan job[T]
	done                chan struct{}
	once                sync.Once
	closed              atomic.Bool
	currentJob          atomic.Pointer[job[T]] // tracks job being executed for panic notification
	contentionWarnAfter time.Duration          // threshold for queue wait warning (default 1s)
}

// DefaultContentionThreshold is the default duration after which a queued job
// triggers a contention warning log. Jobs waiting longer than this threshold
// indicate the worker is overloaded or upstream jobs are taking too long.
const DefaultContentionThreshold = time.Second

func NewJobQueue[T any](queueSize ...int) *JobQueue[T] {

	size := 100
	if len(queueSize) > 0 {
		size = queueSize[0]
	}

	q := &JobQueue[T]{
		jobs:                make(chan job[T], size),
		done:                make(chan struct{}),
		contentionWarnAfter: DefaultContentionThreshold,
	}

	go q.runWorker()

	return q
}

// SetContentionThreshold sets the duration after which a queued job triggers
// a contention warning. Set to 0 or negative to disable contention warnings.
// Must be called before submitting jobs (not concurrency-safe).
func (q *JobQueue[T]) SetContentionThreshold(d time.Duration) {
	q.contentionWarnAfter = d
}

// Close drains the queue and shuts down the worker goroutine. Safe to call
// multiple times; only the first call has any effect.
func (q *JobQueue[T]) Close() {
	q.once.Do(func() {
		q.closed.Store(true)
		close(q.jobs)
	})
	<-q.done
}

func (q *JobQueue[T]) runWorker() {
	defer close(q.done)

	for {
		if cleanExit := q.runWorkerOnce(); cleanExit {
			return
		}
		log.Println("[WARNING] job worker restarted after panic")
	}
}

// runWorkerOnce runs the worker loop and returns true on a clean exit (jobs
// channel closed) or false if the worker panicked.
func (q *JobQueue[T]) runWorkerOnce() (cleanExit bool) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Printf(
				"[PANIC] job worker recovered: %v\n%s",
				r,
				string(buf[:n]),
			)

			// Notify blocked caller if there was a job in progress
			if j := q.currentJob.Swap(nil); j != nil {
				var zero T
				select {
				case j.resp <- result[T]{zero, fmt.Errorf("worker panic: %v", r)}:
				default:
					// resp channel full or caller already gone
				}
			}
			// cleanExit remains false (zero value)
		}
	}()

	q.worker()
	return true
}

func (q *JobQueue[T]) worker() {

	for j := range q.jobs {

		wait := time.Since(j.enqueuedAt)

		if q.contentionWarnAfter > 0 && wait > q.contentionWarnAfter {
			log.Printf(
				"[WARNING] job queue contention: waited %v - %s - %s:%d",
				wait,
				j.contextInfo,
				j.callerFile,
				j.callerLine,
			)
		}

		// Note: if a job's context is already done when dequeued, we still
		// dequeue it — we cannot put it back. The job is skipped but the queue
		// slot was already consumed. This is an inherent property of a
		// single-worker queue.
		select {
		case <-j.ctx.Done():
			var zero T
			select {
			case j.resp <- result[T]{zero, j.ctx.Err()}:
			case <-j.ctx.Done():
			}
			continue
		default:
		}

		// Track current job for panic notification (cleared after execution)
		jCopy := j
		q.currentJob.Store(&jCopy)

		start := time.Now()

		res, err := safeExecute(j.fn)

		q.currentJob.Store(nil)

		duration := time.Since(start)

		if ctxErr := j.ctx.Err(); ctxErr != nil {
			logContextError(
				j.operationName,
				ctxErr,
				duration,
				j.contextInfo,
				j.callerFile,
				j.callerLine,
			)
		}

		select {
		case j.resp <- result[T]{res, err}:
		case <-j.ctx.Done():
			if err != nil {
				log.Printf(
					"[WARNING] %s result discarded (context expired): %v - %s:%d",
					j.operationName,
					err,
					j.callerFile,
					j.callerLine,
				)
			}
		}
	}
}

func safeExecute[T any](fn func() (T, error)) (res T, err error) {

	defer func() {
		if r := recover(); r != nil {

			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)

			log.Printf(
				"[PANIC] job execution panic: %v\n%s",
				r,
				string(buf[:n]),
			)

			err = fmt.Errorf("job panic: %v", r)
		}
	}()

	return fn()
}

func (q *JobQueue[T]) Exec(operationName string, fn func() (T, error)) (T, error) {
	return q.execInternal(2, context.Background(), operationName, "", fn)
}

// ExecWithTimeout enqueues a function to be executed by the worker and waits
// for the result, with a timeout. If the timeout expires before the job is
// enqueued or before a result is received, a context deadline exceeded error
// is returned.
//
// IMPORTANT: Timeout expiration does NOT prevent execution of an already-
// enqueued job. Once a job is in the queue, it will be executed by the worker
// (though the result may be discarded if the timeout has expired).
// Callers must design their functions to be safe for execution even after
// timeout, or accept that side effects may still occur.
func (q *JobQueue[T]) ExecWithTimeout(
	timeout time.Duration,
	operationName string,
	contextInfo string,
	fn func() (T, error),
) (T, error) {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return q.execInternal(2, ctx, operationName, contextInfo, fn)
}

// ExecWithContext enqueues a function to be executed by the worker and waits
// for the result. If the context is cancelled before the job is enqueued or
// before a result is received, the context error is returned.
//
// IMPORTANT: Context cancellation does NOT prevent execution of an already-
// enqueued job. Once a job is in the queue, it will be executed by the worker
// (though the result may be discarded if the caller's context has expired).
// Callers must design their functions to be safe for execution even after
// context cancellation, or accept that side effects may still occur.
func (q *JobQueue[T]) ExecWithContext(
	ctx context.Context,
	operationName string,
	contextInfo string,
	fn func() (T, error),
) (T, error) {

	return q.execInternal(2, ctx, operationName, contextInfo, fn)
}

func (q *JobQueue[T]) execInternal(
	callerDepth int,
	ctx context.Context,
	operationName string,
	contextInfo string,
	fn func() (T, error),
) (T, error) {

	if q.closed.Load() {
		var zero T
		return zero, ErrQueueClosed
	}

	resp := make(chan result[T], 1)

	_, callerFile, callerLine, _ := runtime.Caller(callerDepth)

	j := job[T]{
		ctx:           ctx,
		operationName: operationName,
		contextInfo:   contextInfo,
		fn:            fn,
		resp:          resp,
		enqueuedAt:    time.Now(),
		callerFile:    callerFile,
		callerLine:    callerLine,
	}

	if sent, err := q.trySend(ctx, j); !sent {
		var zero T
		return zero, err
	}

	select {

	case r := <-resp:
		return r.val, r.err

	case <-ctx.Done():

		var zero T

		return zero, fmt.Errorf("%s: %w", operationName, ctx.Err())
	}
}

// trySend attempts to send a job to the queue. Returns (true, nil) on success,
// (false, ctx.Err()) if context cancelled, or (false, ErrQueueClosed) if the
// channel was closed (caught via recover from send-on-closed-channel panic).
func (q *JobQueue[T]) trySend(ctx context.Context, j job[T]) (sent bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			// send on closed channel
			sent = false
			err = ErrQueueClosed
		}
	}()

	select {
	case q.jobs <- j:
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// Len returns the approximate number of jobs currently waiting in the queue.
// This is a point-in-time snapshot and may be stale by the time it is used.
func (q *JobQueue[T]) Len() int {
	return len(q.jobs)
}

// Capacity returns the maximum number of jobs the queue can buffer before
// blocking on enqueue (or returning context error if context expires first).
func (q *JobQueue[T]) Capacity() int {
	return cap(q.jobs)
}

func logContextError(
	operationName string,
	ctxErr error,
	duration time.Duration,
	contextInfo string,
	callerFile string,
	callerLine int,
) {
	log.Printf(
		"[CONTEXT] %s context error after %v: %v\n"+
			"  Context: %s\n"+
			"  Caller: %s:%d",
		operationName,
		duration,
		ctxErr,
		contextInfo,
		callerFile,
		callerLine,
	)
}
