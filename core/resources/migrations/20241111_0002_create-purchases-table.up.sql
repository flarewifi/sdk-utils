CREATE TABLE IF NOT EXISTS purchases (
    id INTEGER PRIMARY KEY,
    uid VARCHAR(36) NOT NULL UNIQUE,
    device_id INTEGER NOT NULL,
    sku VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(8, 2) NOT NULL,
    any_price BOOLEAN NOT NULL DEFAULT FALSE,
    callback_plugin VARCHAR(255) NOT NULL,
    callback_route VARCHAR(510) NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',

    wallet_debit DECIMAL(8, 2) NOT NULL DEFAULT 0.0,
    wallet_tx_id INTEGER DEFAULT NULL,

    confirmed_at TIMESTAMP NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_purchases_device_id ON purchases(device_id);
