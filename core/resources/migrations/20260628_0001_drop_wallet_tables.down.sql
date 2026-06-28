-- Reverse of drop_wallet_tables: restore the dropped tables and re-add the
-- removed purchases columns.

-- 1. Recreate wallets.
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

-- 2. Recreate wallet_transactions.
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    wallet_id INTEGER NOT NULL,
    amount DECIMAL(8, 2) NOT NULL,
    new_balance DECIMAL(8, 2) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (wallet_id) REFERENCES wallets (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE UNIQUE INDEX IF NOT EXISTS index_wallet_transaction_uuid ON wallet_transactions(uuid) WHERE uuid != '';

-- 3. Re-add the removed columns to purchases.
ALTER TABLE purchases ADD COLUMN wallet_debit DECIMAL(8, 2) NOT NULL DEFAULT 0.0;
ALTER TABLE purchases ADD COLUMN wallet_tx_id INTEGER DEFAULT NULL;
