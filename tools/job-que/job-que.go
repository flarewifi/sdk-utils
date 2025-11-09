package jobque

// Exec runs the given functions in series, waiting for each to complete before starting the next.
import "sync"

// JobQue serializes function execution using a mutex.
type JobQue[T any] struct {
	mu sync.Mutex
}

func NewJobQue[T any]() *JobQue[T] {
	return &JobQue[T]{}
}

// Exec runs the given function in a serialized manner using the JobQue's mutex.
func (q *JobQue[T]) Exec(fn func() (T, error)) (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return fn()
}
