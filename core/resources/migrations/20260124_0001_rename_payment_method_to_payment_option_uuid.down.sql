-- Revert: Rename payment_option_uuid back to payment_method
ALTER TABLE payments RENAME COLUMN payment_option_uuid TO payment_method;

-- Recreate the original index
DROP INDEX IF EXISTS index_payment_option_uuid;
CREATE INDEX IF NOT EXISTS index_payment_method ON payments(payment_method);
