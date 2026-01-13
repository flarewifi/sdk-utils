-- Rollback: Remove timezone column and index
DROP INDEX IF EXISTS idx_fingerprints_timezone;
ALTER TABLE device_fingerprints DROP COLUMN timezone;
