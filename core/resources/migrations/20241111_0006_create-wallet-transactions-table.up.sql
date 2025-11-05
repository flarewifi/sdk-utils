CREATE TABLE IF NOT EXISTS wallet_transactions (
    id INTEGER PRIMARY KEY,
    wallet_id INTEGER NOT NULL,
    amount DECIMAL(8, 2) NOT NULL,
    new_balance DECIMAL(8, 2) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (wallet_id) REFERENCES wallets (id) ON DELETE CASCADE
);
