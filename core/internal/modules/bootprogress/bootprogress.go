// Package bootprogress holds the live, thread-safe state the boot sequence
// publishes so the booting web page (served by the BootingRouter) can render what
// the machine is doing while it comes up — most importantly the online-wait and
// system_packages install phases, which can take minutes.
//
// It is intentionally a leaf package (only stdlib): the boot goroutine in
// core/internal/boot writes it, while the /boot/progress handler in
// core/internal/web/controllers reads it. Since boot imports web (init-http.go),
// web cannot import boot; both reach this tracker through CoreGlobals instead,
// which avoids an import cycle.
package bootprogress

import "sync"

// StepStatus is the lifecycle of a single boot milestone shown on the page.
type StepStatus string

const (
	// StatusActive marks the milestone currently in progress (rendered with a
	// spinner/ellipsis by the booting page).
	StatusActive StepStatus = "active"
	// StatusDone marks a completed milestone (rendered with a checkmark).
	StatusDone StepStatus = "done"
)

// Step is one milestone in the boot timeline. Current/Total are non-zero only for
// counted phases (e.g. installing packages for plugin 2 of 5); the page formats
// the count itself so the translated Label stays free of interpolated numbers.
type Step struct {
	Label   string     `json:"label"`
	Status  StepStatus `json:"status"`
	Current int        `json:"current,omitempty"`
	Total   int        `json:"total,omitempty"`
}

// Snapshot is an immutable copy of the timeline, safe to serialize to the page.
type Snapshot struct {
	Steps []Step `json:"steps"`
}

// Tracker is the concurrency-safe boot timeline. The zero value is not usable;
// construct it with New.
type Tracker struct {
	mu    sync.RWMutex
	steps []Step
}

// New returns an empty tracker.
func New() *Tracker {
	return &Tracker{}
}

// Advance completes the current active step (if any) and starts a new active step
// with the given label. Calling it for each milestone produces the running
// checklist the booting page renders.
func (t *Tracker) Advance(label string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.completeLastLocked()
	t.steps = append(t.steps, Step{Label: label, Status: StatusActive})
}

// SetActiveProgress records a count (e.g. "2 of 5") on the current active step.
// It is a no-op when there is no active step. The count is kept separate from the
// label so translations need not interpolate numbers.
func (t *Tracker) SetActiveProgress(current, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if n := len(t.steps); n > 0 && t.steps[n-1].Status == StatusActive {
		t.steps[n-1].Current = current
		t.steps[n-1].Total = total
	}
}

// Done marks the current active step complete. Called once at the end of boot so
// the final milestone is checked off before the page redirects to the app.
func (t *Tracker) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.completeLastLocked()
}

// Snapshot returns an independent copy of the timeline for serialization.
func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]Step, len(t.steps))
	copy(out, t.steps)
	return Snapshot{Steps: out}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// completeLastLocked flips a trailing active step to done. Callers must hold the
// write lock.
func (t *Tracker) completeLastLocked() {
	if n := len(t.steps); n > 0 && t.steps[n-1].Status == StatusActive {
		t.steps[n-1].Status = StatusDone
	}
}
