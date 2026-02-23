DROP INDEX IF EXISTS idx_vouchers_batch_uuid;
ALTER TABLE vouchers DROP COLUMN batch_uuid;
