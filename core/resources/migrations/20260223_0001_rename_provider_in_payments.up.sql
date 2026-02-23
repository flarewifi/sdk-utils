-- Rename provider_pkg to provider and remove provider_name
ALTER TABLE payments RENAME COLUMN provider_pkg TO provider;
ALTER TABLE payments DROP COLUMN provider_name;

-- Recreate index with new column name
DROP INDEX IF EXISTS index_payments_provider_pkg;
CREATE INDEX IF NOT EXISTS index_payments_provider ON payments(provider);
