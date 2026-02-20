-- Add uuid column to vouchers table for cloud sync
ALTER TABLE vouchers ADD COLUMN uuid VARCHAR(36) NOT NULL DEFAULT '';

-- Create unique index on uuid (only for non-empty values)
CREATE UNIQUE INDEX idx_vouchers_uuid ON vouchers(uuid) WHERE uuid != '';
