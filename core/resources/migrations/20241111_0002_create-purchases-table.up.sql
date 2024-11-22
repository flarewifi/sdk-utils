CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    sku VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(8, 2) NOT NULL,
    any_price BOOLEAN NOT NULL DEFAULT FALSE,
    callback_plugin VARCHAR(255) NOT NULL,
    callback_vue_route_name VARCHAR(2048),

    wallet_debit DECIMAL(8, 2) NOT NULL DEFAULT 0.0,
    wallet_tx_id UUID DEFAULT NULL,

    confirmed_at TIMESTAMP NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);
