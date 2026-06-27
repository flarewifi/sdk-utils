-- Drop the unused wallet tables and remove the wallet_debit / wallet_tx_id
-- columns from purchases.
-- NOTE: The migration runner wraps each file in its own transaction - no BEGIN/COMMIT needed here.
-- NOTE: SQLite (older versions) cannot DROP COLUMN, so purchases is recreated.

-- 1. Drop the tables (wallet_transactions first - it has a FK to wallets).
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS wallets;

-- 2. Recreate purchases without the wallet columns.
CREATE TABLE purchases_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    device_id INTEGER,  -- nullable for admin purchases (e.g., voucher batch sales)
    sku VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(8, 2) NOT NULL,
    any_price BOOLEAN NOT NULL DEFAULT FALSE,
    callback_plugin VARCHAR(255) NOT NULL,
    callback_route VARCHAR(510) NOT NULL,
    webhook_route VARCHAR(510) NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',

    processing BOOLEAN NOT NULL DEFAULT FALSE,
    payment_url TEXT NOT NULL DEFAULT '',
    payment_note TEXT NOT NULL DEFAULT '',

    confirmed_at TIMESTAMP NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

INSERT INTO purchases_new (
    id, uuid, device_id, sku, name, description, price, any_price,
    callback_plugin, callback_route, webhook_route, metadata,
    processing, payment_url, payment_note,
    confirmed_at, cancelled_at, cancelled_reason, created_at
)
SELECT
    id, uuid, device_id, sku, name, description, price, any_price,
    callback_plugin, callback_route, webhook_route, metadata,
    processing, payment_url, payment_note,
    confirmed_at, cancelled_at, cancelled_reason, created_at
FROM purchases;

DROP TABLE purchases;
ALTER TABLE purchases_new RENAME TO purchases;

CREATE INDEX IF NOT EXISTS index_purchases_device_id ON purchases(device_id);
