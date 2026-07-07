/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"
)

// ISchedulerApi lets a plugin run background work that is tracked and
// automatically stopped during graceful shutdown. Prefer this over a bare
// `go func(){ ... }()` or a hand-rolled `time.NewTicker` loop for anything
// long-running or periodic — those have no way to be told to stop.
type ISchedulerApi interface {
	// Go starts fn in a supervised goroutine. fn must return promptly when
	// ctx is cancelled (on shutdown, or an explicit Cancel(name)). name must
	// be unique within this plugin — registering a duplicate name returns an
	// error rather than silently ignoring it or replacing the running task.
	Go(name string, fn func(ctx context.Context)) error

	// Every runs fn once immediately and then every interval, until ctx is
	// cancelled. A panic inside fn is recovered and logged; it stops that
	// task only, not the process — plugins share one process, so an
	// unrecovered panic here would otherwise take everything down.
	Every(name string, interval time.Duration, fn func(ctx context.Context)) error

	// Cron runs fn on a standard 5-field cron expression (minute hour
	// day-of-month month day-of-week — e.g. "0 3 * * *" for daily at 3am, in
	// the machine's local time zone), until ctx is cancelled. Returns an
	// error immediately if expr fails to parse, so a typo'd expression fails
	// fast instead of silently never firing. Panics are recovered and logged
	// the same as Every.
	Cron(name string, expr string, fn func(ctx context.Context)) error

	// Cancel stops a previously registered task by name. No-op if the name
	// is unknown or the task already stopped on its own.
	Cancel(name string)
}
