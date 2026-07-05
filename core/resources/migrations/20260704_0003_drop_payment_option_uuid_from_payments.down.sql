ALTER TABLE payments ADD COLUMN payment_option_uuid VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS index_payment_option_uuid ON payments(payment_option_uuid);
