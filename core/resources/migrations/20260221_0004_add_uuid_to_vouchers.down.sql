-- Remove uuid column from vouchers table
DROP INDEX IF EXISTS idx_vouchers_uuid;
ALTER TABLE vouchers DROP COLUMN uuid;
