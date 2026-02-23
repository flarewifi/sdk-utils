-- Replace voucher_batches table with new schema
-- Consolidates: drop_payment_fields, drop_voucher_batches, create_voucher_batches,
--               make_amount_nullable, add_provider_pkg

-- Drop old table if exists
DROP INDEX IF EXISTS idx_voucher_batches_uuid;
DROP TABLE IF EXISTS voucher_batches;

-- Create new voucher_batches table
CREATE TABLE voucher_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,
    provider_pkg TEXT NOT NULL DEFAULT '',
    amount REAL,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_voucher_batches_uuid ON voucher_batches(uuid);
CREATE INDEX idx_voucher_batches_provider_pkg ON voucher_batches(provider_pkg);
