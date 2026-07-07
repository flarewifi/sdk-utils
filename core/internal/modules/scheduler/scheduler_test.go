package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordingLogger collects Error() calls so panic-recovery can be asserted on.
type recordingLogger struct {
	mu     sync.Mutex
	errors []string
}

func (l *recordingLogger) Info(string) error  { return nil }
func (l *recordingLogger) Debug(string) error { return nil }
func (l *recordingLogger) Error(message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, message)
	return nil
}

func (l *recordingLogger) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.errors)
}

func TestGoRunsAndStopsOnShutdown(t *testing.T) {
	m := NewManager()

	var ran atomic.Bool
	var cancelled atomic.Bool

	if err := m.Go("owner", "task", func(ctx context.Context) {
		ran.Store(true)
		<-ctx.Done()
		cancelled.Store(true)
	}); err != nil {
		t.Fatalf("Go returned error: %v", err)
	}

	// Give the goroutine a moment to start and block on ctx.Done().
	time.Sleep(20 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("task never ran")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if !cancelled.Load() {
		t.Fatal("task's ctx was never cancelled by Shutdown")
	}
}

func TestGoDuplicateNameRejected(t *testing.T) {
	m := NewManager()

	block := make(chan struct{})
	if err := m.Go("owner", "dup", func(ctx context.Context) { <-block }); err != nil {
		t.Fatalf("first Go returned error: %v", err)
	}

	err := m.Go("owner", "dup", func(ctx context.Context) {})
	if err == nil {
		t.Fatal("expected an error registering a duplicate name, got nil")
	}

	// A different owner may reuse the same name without colliding.
	if err := m.Go("other-owner", "dup", func(ctx context.Context) {}); err != nil {
		t.Fatalf("expected a different owner to reuse the name freely, got: %v", err)
	}

	close(block)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestEveryTicksAndStops(t *testing.T) {
	m := NewManager()

	var ticks atomic.Int32
	if err := m.Every("owner", "tick", 10*time.Millisecond, func(ctx context.Context) {
		ticks.Add(1)
	}); err != nil {
		t.Fatalf("Every returned error: %v", err)
	}

	time.Sleep(55 * time.Millisecond)
	m.Cancel("owner", "tick")

	// Let the cancellation land, then snapshot the tick count.
	time.Sleep(20 * time.Millisecond)
	after := ticks.Load()
	if after < 3 {
		t.Fatalf("expected at least 3 ticks (1 immediate + periodic), got %d", after)
	}

	time.Sleep(30 * time.Millisecond)
	if ticks.Load() != after {
		t.Fatalf("task kept ticking after Cancel: had %d, now %d", after, ticks.Load())
	}
}

func TestCronInvalidExpressionRejected(t *testing.T) {
	m := NewManager()

	if err := m.Cron("owner", "bad-cron", "not a cron expression", func(ctx context.Context) {}); err == nil {
		t.Fatal("expected an error for an invalid cron expression, got nil")
	}
}

func TestCronRunsOnSchedule(t *testing.T) {
	m := NewManager()

	// Every minute — combined with the immediate first run, this only proves
	// registration + the immediate run, not the parsed schedule's Next() math,
	// which TestCronInvalidExpressionRejected and the ParseStandard library
	// itself already cover. Assert the immediate run happens.
	var ran atomic.Bool
	if err := m.Cron("owner", "every-minute", "* * * * *", func(ctx context.Context) {
		ran.Store(true)
	}); err != nil {
		t.Fatalf("Cron returned error: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("cron task never ran its immediate first invocation")
	}

	m.Cancel("owner", "every-minute")
}

func TestPanicIsRecoveredAndLogged(t *testing.T) {
	m := NewManager()
	logger := &recordingLogger{}
	m.SetLogger(logger)

	done := make(chan struct{})
	if err := m.Go("owner", "panicker", func(ctx context.Context) {
		defer close(done)
		panic(errors.New("boom"))
	}); err != nil {
		t.Fatalf("Go returned error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("panicking task never ran")
	}

	// Give runOnce's recover a moment to log before asserting.
	time.Sleep(20 * time.Millisecond)
	if logger.count() != 1 {
		t.Fatalf("expected exactly 1 logged panic, got %d", logger.count())
	}

	// The panic must not have crashed the test process (if it had, we
	// wouldn't reach here) and Shutdown must still complete cleanly since the
	// panicking goroutine unregisters itself via its deferred unregister call.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestEveryStopsLoopingAfterPanic(t *testing.T) {
	m := NewManager()
	m.SetLogger(&recordingLogger{})

	var calls atomic.Int32
	if err := m.Every("owner", "flaky", 10*time.Millisecond, func(ctx context.Context) {
		calls.Add(1)
		panic("first tick always panics")
	}); err != nil {
		t.Fatalf("Every returned error: %v", err)
	}

	time.Sleep(60 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected the loop to stop after its first panicking tick, got %d calls", got)
	}
}

func TestShutdownTimesOutWithRunningTask(t *testing.T) {
	m := NewManager()

	release := make(chan struct{})
	if err := m.Go("owner", "stuck", func(ctx context.Context) {
		// Deliberately ignores ctx cancellation to exercise the timeout path.
		<-release
	}); err != nil {
		t.Fatalf("Go returned error: %v", err)
	}
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := m.Shutdown(ctx)
	if err == nil {
		t.Fatal("expected Shutdown to time out with the stuck task still running")
	}
}

func TestGoAfterShutdownRejected(t *testing.T) {
	m := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	if err := m.Go("owner", "too-late", func(ctx context.Context) {}); err == nil {
		t.Fatal("expected Go to reject registration once the Manager is shutting down")
	}
}

// TestRegisterDuringShutdownRace exercises the exact race Shutdown's
// shuttingDown flag guards against: registrations landing concurrently with
// the WaitGroup's Wait. Every register (Go/Every/Cron all funnel through it)
// must either fully reserve its WaitGroup slot before Shutdown flips the
// flag, or be rejected outright — never Add after Wait has already looked at
// a zero counter. Run with -race; a regression here either panics
// ("WaitGroup misuse") or trips the race detector.
func TestRegisterDuringShutdownRace(t *testing.T) {
	m := NewManager()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = m.Go("owner", fmt.Sprintf("task-%d", i), func(ctx context.Context) {
				<-ctx.Done()
			})
		}(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = m.Shutdown(ctx)

	wg.Wait()
}
