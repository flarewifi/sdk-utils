CREATE TABLE IF NOT EXISTS voucher_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,
    total_amount REAL,
    payment_note TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_voucher_batches_uuid ON voucher_batches(uuid);
