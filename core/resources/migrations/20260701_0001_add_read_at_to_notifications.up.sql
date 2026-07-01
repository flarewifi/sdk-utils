-- Track WHEN a notification was read so the cleanup job can delete notifications
-- that have been read for more than 30 days. NULL = unread (or never read).
-- Read state itself stays in `status` (0 = unread, 1 = read); read_at is only the
-- timestamp of the transition to read, cleared if a notification is marked unread.
ALTER TABLE notifications ADD COLUMN read_at TIMESTAMP;

-- Partial-ish index to make the age sweep (status = 1 AND read_at < cutoff) cheap.
CREATE INDEX IF NOT EXISTS index_notifications_read_at ON notifications(read_at);
