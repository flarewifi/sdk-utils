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
// Indent marks a child line nested under the preceding top-level step (e.g. each
// plugin listed under "Loading plugins"); the page renders it indented.
type Step struct {
	Label   string     `json:"label"`
	Status  StepStatus `json:"status"`
	Current int        `json:"current,omitempty"`
	Total   int        `json:"total,omitempty"`
	Indent  bool       `json:"indent,omitempty"`
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

// Advance starts a new top-level phase. It completes ALL still-active steps first
// — both the previous phase and any child left active under it — so a parent phase
// is ticked only once the next phase begins (i.e. after all its children finished).
func (t *Tracker) Advance(label string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.completeAllActiveLocked()
	t.steps = append(t.steps, Step{Label: label, Status: StatusActive})
}

// Substep starts a new active child step, indented under the current phase. It
// completes only the previous SIBLING child (the last active indented step) and
// deliberately leaves the parent phase active, so the parent stays "in progress"
// until every child is done and the next Advance/Done ticks it off.
func (t *Tracker) Substep(label string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if n := len(t.steps); n > 0 && t.steps[n-1].Status == StatusActive && t.steps[n-1].Indent {
		t.steps[n-1].Status = StatusDone
	}
	t.steps = append(t.steps, Step{Label: label, Status: StatusActive, Indent: true})
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

// Done marks all still-active steps complete. Called once at the end of boot so
// the final phase (and any child left active under it) is checked off before the
// page redirects to the app.
func (t *Tracker) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.completeAllActiveLocked()
}

// Snapshot returns an independent copy of the timeline for serialization.
func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]Step, len(t.steps))
	copy(out, t.steps)
	return Snapshot{Steps: out}
}

// Release drops the recorded timeline so the GC can reclaim it. Call it once boot
// is complete and the booting HTTP server has shut down — at that point /boot/
// progress is gone and nothing reads the steps again, yet the Tracker is still
// reachable via CoreGlobals for the whole process lifetime. The checklist can hold
// a couple dozen Steps (each plugin appears under both the "Compiling plugins" and
// "Loading plugins" phases), so releasing it returns that memory for good. After
// Release, Snapshot returns an empty timeline; the zero-length slice keeps any
// late, stray call safe rather than panicking.
func (t *Tracker) Release() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.steps = nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// completeAllActiveLocked flips every still-active step to done. Used when a phase
// ends (Advance/Done) so both the phase and any child left active under it are
// ticked together. In a well-formed sequence the only active steps are the current
// phase and its last child, so this never touches an already-finished earlier
// phase. Callers must hold the write lock.
func (t *Tracker) completeAllActiveLocked() {
	for i := range t.steps {
		if t.steps[i].Status == StatusActive {
			t.steps[i].Status = StatusDone
		}
	}
}
