CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id UUID NOT NULL,
    amount DECIMAL(8, 2) DEFAULT 0.0,
    optname VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (purchase_id) REFERENCES purchases (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_optname ON payments(optname);
