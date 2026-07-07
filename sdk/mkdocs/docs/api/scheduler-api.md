# ISchedulerApi

The `ISchedulerApi` runs background work — long-running goroutines and scheduled/periodic tasks — that is tracked by the machine and stopped automatically on graceful shutdown. Prefer this over a bare `go func(){ ... }()` or a hand-rolled `time.NewTicker` loop for anything long-running or periodic in a plugin: those have no way to be told to stop, and a panic inside them can crash the whole machine process (plugins all run in one process).

## Interface Definition

```go
type ISchedulerApi interface {
    Go(name string, fn func(ctx context.Context)) error
    Every(name string, interval time.Duration, fn func(ctx context.Context)) error
    Cron(name string, expr string, fn func(ctx context.Context)) error
    Cancel(name string)
}
```

## Methods

| Method | Description |
| ---- | ---- |
| `Go(name string, fn func(ctx context.Context)) error` | Starts `fn` in a supervised goroutine. `fn` must return promptly when `ctx` is cancelled (on shutdown, or `Cancel(name)`). |
| `Every(name string, interval time.Duration, fn func(ctx context.Context)) error` | Runs `fn` once immediately, then every `interval`, until `ctx` is cancelled. |
| `Cron(name string, expr string, fn func(ctx context.Context)) error` | Runs `fn` on a standard 5-field cron expression (minute hour day-of-month month day-of-week, e.g. `"0 3 * * *"` for daily at 3am), in the machine's local time zone, until `ctx` is cancelled. |
| `Cancel(name string)` | Stops a previously registered task by name. No-op if the name is unknown or the task already stopped on its own. |

`name` must be unique within your plugin — registering a duplicate name returns an error rather than silently ignoring it or replacing the running task. Two different plugins can each register a task called the same thing without colliding; names are scoped per plugin internally.

A panic inside any registered `fn` is recovered and logged (via [`ILoggerApi`](logger-api.md)) instead of crashing the machine. For `Every`/`Cron`, a panic stops that task rather than silently continuing to tick against whatever broken state the closure may now be in — register the task again (e.g. after fixing the underlying issue and restarting the plugin) if it needs to resume.

## Usage Examples

### Long-running background worker

```go
api.Scheduler().Go("sync-worker", func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(5 * time.Second):
            doSync()
        }
    }
})
```

### One-shot background task

`Go` doesn't require a loop — it's just a supervised, panic-protected goroutine, so a single piece of background work that runs once and returns is just as valid. This is useful for something slow you don't want to block your plugin's `Init` on, but still want tracked (and cut short if shutdown happens while it's still running).

With no loop, there's nowhere for a `select` on `ctx.Done()` to live, so cancellation has to be handled one of two ways instead:

**The work is naturally interruptible** — pass `ctx` down into whatever I/O you're doing (a DB query, an HTTP request) and let *that* honor cancellation. This is the common case and needs no explicit check of your own:

```go
api.Scheduler().Go("initial-sync", func(ctx context.Context) {
    rows, err := api.SqlDB().QueryContext(ctx, "SELECT ... FROM historical_records")
    if err != nil {
        // Cancellation surfaces here as ctx.Err() (context.Canceled) — same
        // as any other query error, nothing special to unwrap.
        return
    }
    defer rows.Close()
    processRows(rows)
})
```

**The work is a sequence of discrete steps with no single context-aware call to delegate to** — check `ctx.Err()` between steps instead of wrapping the whole thing in a `select`:

```go
api.Scheduler().Go("initial-sync", func(ctx context.Context) {
    for _, batch := range historicalBatches() {
        if ctx.Err() != nil {
            return // shutdown requested — stop before starting the next batch
        }
        processBatch(batch)
    }
})
```

If the work is a single blocking call that accepts no `context.Context` at all, it genuinely can't be interrupted mid-call — `Shutdown` will wait for it to finish on its own, up to its timeout, same as it would for any other slow task.

### Periodic interval task

```go
api.Scheduler().Every("cleanup", 1*time.Hour, func(ctx context.Context) {
    cleanupExpiredRecords(ctx)
})
```

### Cron-scheduled task

```go
api.Scheduler().Cron("nightly-report", "0 3 * * *", func(ctx context.Context) {
    generateNightlyReport(ctx)
})
```

### Cancelling a task early

```go
api.Scheduler().Cancel("sync-worker")
```

## Graceful Shutdown

Every task registered through `ISchedulerApi` is cancelled automatically when the machine process receives a shutdown signal, and the shutdown sequence waits (up to a bounded timeout) for all registered tasks to return before the process exits. Plugins don't need to (and can't) hook into this directly — just make sure your `fn` actually returns when `ctx` is cancelled, rather than ignoring it.

### Flushing in-flight work on shutdown

A task that accumulates state between ticks (e.g. batching writes) should treat `ctx.Done()` as "flush now, then return" rather than just returning immediately and losing whatever hasn't been persisted since the last tick:

```go
api.Scheduler().Go("usage-batcher", func(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            // ctx is already cancelled at this point, so it can't be reused
            // for the flush itself — a DB call or HTTP request given a
            // cancelled context fails immediately. Use a fresh, short-lived
            // context for the final write instead.
            flushCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()
            flushUsage(flushCtx)
            return
        case <-ticker.C:
            flushUsage(ctx)
        }
    }
})
```

Keep the final flush fast and bounded. `Shutdown` waits for every registered task across every plugin up to one fixed timeout for the whole machine — a slow or hanging cleanup delays every other task's shutdown too, and may get cut off before it finishes anyway.
