ALTER TABLE vouchers ADD COLUMN batch_uuid TEXT;

CREATE INDEX idx_vouchers_batch_uuid ON vouchers(batch_uuid);
