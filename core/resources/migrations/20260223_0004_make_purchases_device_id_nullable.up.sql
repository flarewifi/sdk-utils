-- Make device_id nullable for admin-generated purchases (e.g., voucher batch sales)
-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table

CREATE TABLE purchases_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    device_id INTEGER,  -- Now nullable for admin purchases
    sku VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(8, 2) NOT NULL,
    any_price BOOLEAN NOT NULL DEFAULT FALSE,
    callback_plugin VARCHAR(255) NOT NULL,
    callback_route VARCHAR(510) NOT NULL,
    webhook_route VARCHAR(510) NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',

    wallet_debit DECIMAL(8, 2) NOT NULL DEFAULT 0.0,
    wallet_tx_id INTEGER DEFAULT NULL,

    processing BOOLEAN NOT NULL DEFAULT FALSE,
    payment_url TEXT NOT NULL DEFAULT '',
    payment_note TEXT NOT NULL DEFAULT '',

    confirmed_at TIMESTAMP NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

-- Copy existing data
INSERT INTO purchases_new (
    id, uuid, device_id, sku, name, description, price, any_price,
    callback_plugin, callback_route, webhook_route, metadata,
    wallet_debit, wallet_tx_id, processing, payment_url, payment_note,
    confirmed_at, cancelled_at, cancelled_reason, created_at
)
SELECT 
    id, uuid, device_id, sku, name, description, price, any_price,
    callback_plugin, callback_route, webhook_route, metadata,
    wallet_debit, wallet_tx_id, processing, payment_url, payment_note,
    confirmed_at, cancelled_at, cancelled_reason, created_at
FROM purchases;

-- Drop old table and rename new one
DROP TABLE purchases;
ALTER TABLE purchases_new RENAME TO purchases;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS index_purchases_device_id ON purchases(device_id);
