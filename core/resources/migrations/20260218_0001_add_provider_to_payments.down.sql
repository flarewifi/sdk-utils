-- Drop index first
DROP INDEX IF EXISTS index_payments_provider_pkg;

-- Remove provider columns
ALTER TABLE payments DROP COLUMN provider_pkg;
ALTER TABLE payments DROP COLUMN provider_name;
