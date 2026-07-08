// Package scheduler provides the shared registry backing sdk/api's
// ISchedulerApi: every plugin's Go/Every/Cron call registers a task here, so a
// single Shutdown can cancel and wait for all of them across every plugin.
package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	sdkapi "sdk/api"
)

// Manager is the process-wide scheduler registry. One instance lives on
// CoreGlobals and is shared by every plugin's per-plugin ISchedulerApi facade.
// The zero value is not usable; construct it with NewManager.
type Manager struct {
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	tasks        map[string]context.CancelFunc
	logger       sdkapi.ILoggerApi
	shuttingDown bool
}

// NewManager returns a Manager ready to accept task registrations. Call
// SetLogger once a logger is available (the Manager is constructed before the
// core plugin API in NewGlobals, so the logger can't be supplied up front).
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
		tasks:  make(map[string]context.CancelFunc),
	}
}

// SetLogger sets the logger used to report panics recovered from task
// functions. Safe to call once, before any tasks that might panic are running.
func (m *Manager) SetLogger(logger sdkapi.ILoggerApi) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
}

// Go starts fn in a supervised goroutine scoped to owner/name. See
// sdkapi.ISchedulerApi.Go for the full contract.
func (m *Manager) Go(owner, name string, fn func(ctx context.Context)) error {
	taskCtx, err := m.register(owner, name)
	if err != nil {
		return err
	}

	go func() {
		defer m.wg.Done()
		defer m.unregister(owner, name)
		m.runOnce(owner, name, taskCtx, fn)
	}()

	return nil
}

// Every runs fn immediately and then every interval. See
// sdkapi.ISchedulerApi.Every for the full contract.
func (m *Manager) Every(owner, name string, interval time.Duration, fn func(ctx context.Context)) error {
	return m.runLoop(owner, name, fn, func(now time.Time) time.Time {
		return now.Add(interval)
	})
}

// Cron runs fn on a standard 5-field cron expression. See
// sdkapi.ISchedulerApi.Cron for the full contract.
func (m *Manager) Cron(owner, name, expr string, fn func(ctx context.Context)) error {
	schedule, err := cron.ParseStandard(expr)
	if err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q: %w", expr, err)
	}

	return m.runLoop(owner, name, fn, schedule.Next)
}

// Cancel stops a previously registered task. See sdkapi.ISchedulerApi.Cancel
// for the full contract.
func (m *Manager) Cancel(owner, name string) {
	m.mu.Lock()
	cancel, ok := m.tasks[key(owner, name)]
	m.mu.Unlock()

	if ok {
		cancel()
	}
}

// Shutdown cancels every registered task and waits for them all to return, up
// to ctx's deadline. It returns an error naming any tasks still running when
// the deadline is hit, so a stuck task is visible in logs rather than just
// silently truncating the shutdown wait.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.shuttingDown = true
	m.mu.Unlock()

	m.cancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		m.mu.Lock()
		remaining := make([]string, 0, len(m.tasks))
		for k := range m.tasks {
			remaining = append(remaining, k)
		}
		m.mu.Unlock()

		return fmt.Errorf("scheduler: shutdown timed out with %d task(s) still running: %s",
			len(remaining), strings.Join(remaining, ", "))
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// register reserves owner/name and returns a context scoped to both this task
// (cancellable individually via Cancel) and the Manager's root context
// (cancelled all at once by Shutdown). Returns an error if the name is
// already in use for this owner, or if the Manager is shutting down.
//
// The WaitGroup slot (wg.Add) is reserved here, under the same lock as the
// shuttingDown check, rather than by the caller after register returns. If
// Add happened outside this lock, a registration racing Shutdown could Add
// after Shutdown's Wait had already observed the counter at zero — Go's
// WaitGroup explicitly forbids that ordering and can panic. Doing both under
// one lock makes registration and "is shutting down" mutually exclusive: a
// task either fully reserves its slot before Shutdown flips the flag, or sees
// the flag and never calls Add at all.
func (m *Manager) register(owner, name string) (context.Context, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shuttingDown {
		return nil, fmt.Errorf("scheduler: manager is shutting down, refusing to register %q", key(owner, name))
	}

	k := key(owner, name)
	if _, exists := m.tasks[k]; exists {
		return nil, fmt.Errorf("scheduler: task %q is already registered", k)
	}

	taskCtx, cancel := context.WithCancel(m.ctx)
	m.tasks[k] = cancel
	m.wg.Add(1)
	return taskCtx, nil
}

// unregister drops owner/name from the registry once its goroutine has
// returned (normally, via panic, or via cancellation), freeing the name for
// reuse and keeping Shutdown's timeout diagnostics accurate.
func (m *Manager) unregister(owner, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, key(owner, name))
}

// runLoop drives both Every and Cron: they differ only in how the next
// wake-up time is computed. A panic recovered from fn stops the loop instead
// of silently continuing to tick against whatever broken state fn's closure
// may now be in.
func (m *Manager) runLoop(owner, name string, fn func(ctx context.Context), next func(now time.Time) time.Time) error {
	taskCtx, err := m.register(owner, name)
	if err != nil {
		return err
	}

	go func() {
		defer m.wg.Done()
		defer m.unregister(owner, name)

		if panicked := m.runOnce(owner, name, taskCtx, fn); panicked {
			return
		}

		for {
			wait := time.Until(next(time.Now()))
			if wait < 0 {
				wait = 0
			}

			timer := time.NewTimer(wait)
			select {
			case <-taskCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
				if panicked := m.runOnce(owner, name, taskCtx, fn); panicked {
					return
				}
			}
		}
	}()

	return nil
}

// runOnce invokes fn once, recovering and logging any panic so it can't take
// down the shared process (plugins all run in-process). Reports whether fn
// panicked so callers can decide whether to keep going.
func (m *Manager) runOnce(owner, name string, ctx context.Context, fn func(ctx context.Context)) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			if logger := m.getLogger(); logger != nil {
				logger.Error(fmt.Sprintf("scheduler: task %q panicked: %v", key(owner, name), r))
			}
		}
	}()

	fn(ctx)
	return false
}

// getLogger returns the current logger under lock, since SetLogger can race
// with a task panicking during startup.
func (m *Manager) getLogger() sdkapi.ILoggerApi {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logger
}

// key namespaces a task name by its owning plugin's package, so two plugins
// can each register e.g. "cleanup" without colliding.
func key(owner, name string) string {
	return owner + "/" + name
}
