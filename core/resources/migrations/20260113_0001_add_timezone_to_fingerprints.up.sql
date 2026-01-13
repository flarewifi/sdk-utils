-- Add timezone column to device_fingerprints table
-- This enables better cookie theft detection and supports multiple browsers per device
ALTER TABLE device_fingerprints ADD COLUMN timezone VARCHAR(10) NOT NULL DEFAULT '';

-- Create index for potential future timezone-based queries
CREATE INDEX IF NOT EXISTS idx_fingerprints_timezone ON device_fingerprints(timezone);
