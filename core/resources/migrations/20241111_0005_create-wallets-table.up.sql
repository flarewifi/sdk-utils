CREATE TABLE IF NOT EXISTS wallets (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL,
    balance DECIMAL(8, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_wallets_device_id ON wallets(device_id);

CREATE INDEX IF NOT EXISTS index_wallets_device_id ON wallets(device_id);
