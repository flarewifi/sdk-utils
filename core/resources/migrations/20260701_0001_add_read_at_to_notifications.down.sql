-- Drop index first
DROP INDEX IF EXISTS index_notifications_read_at;

-- Remove read_at column
ALTER TABLE notifications DROP COLUMN read_at;
