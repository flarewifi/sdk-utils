-- Revert: rename provider back to provider_pkg and add provider_name
ALTER TABLE payments RENAME COLUMN provider TO provider_pkg;
ALTER TABLE payments ADD COLUMN provider_name VARCHAR(255) NOT NULL DEFAULT '';

-- Recreate original index
DROP INDEX IF EXISTS index_payments_provider;
CREATE INDEX IF NOT EXISTS index_payments_provider_pkg ON payments(provider_pkg);
