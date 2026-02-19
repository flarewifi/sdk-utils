-- Add provider columns to track which payment provider processed each payment
-- provider_pkg: Plugin package name (e.g., "com.flarego.wireless-coinslot")
-- provider_name: Display name of the provider (e.g., "Wireless Coinslots")
ALTER TABLE payments ADD COLUMN provider_pkg VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN provider_name VARCHAR(255) NOT NULL DEFAULT '';

-- Index for filtering by provider
CREATE INDEX IF NOT EXISTS index_payments_provider_pkg ON payments(provider_pkg);
