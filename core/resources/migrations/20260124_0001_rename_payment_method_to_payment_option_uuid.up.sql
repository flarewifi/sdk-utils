-- Rename payment_method to payment_option_uuid in payments table
ALTER TABLE payments RENAME COLUMN payment_method TO payment_option_uuid;

-- Recreate the index with new column name
DROP INDEX IF EXISTS index_payment_method;
CREATE INDEX IF NOT EXISTS index_payment_option_uuid ON payments(payment_option_uuid);
