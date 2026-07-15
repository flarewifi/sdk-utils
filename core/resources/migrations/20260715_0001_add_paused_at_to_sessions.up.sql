-- Persist counter-pause state on the session row.
-- When paused_at IS NOT NULL the time/data counters are frozen: elapsed time is
-- NOT added to consumption and the session's remaining time/data stays constant.
-- Was previously an in-memory-only flag (counterPaused); persisting it lets a
-- paused session survive a reboot instead of silently resuming.
ALTER TABLE sessions ADD COLUMN paused_at TIMESTAMP;
