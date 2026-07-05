-- Index for filtering/aggregating sales by payment method
CREATE INDEX IF NOT EXISTS index_payments_payment_method ON payments(payment_method);
