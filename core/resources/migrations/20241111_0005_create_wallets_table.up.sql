CREATE TABLE IF NOT EXISTS wallets (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    device_id INTEGER NOT NULL,
    balance DECIMAL(8, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_wallets_device_id ON wallets(device_id);
CREATE UNIQUE INDEX IF NOT EXISTS index_wallet_uuid ON wallets(uuid) WHERE uuid != '';
