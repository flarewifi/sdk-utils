CREATE TABLE IF NOT EXISTS payments (
    id INTEGER PRIMARY KEY,
    purchase_id INTEGER NOT NULL,
    amount DECIMAL(8, 2) NOT NULL DEFAULT 0.0,
    payment_method VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (purchase_id) REFERENCES purchases (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_peyment_method ON payments(payment_method);
