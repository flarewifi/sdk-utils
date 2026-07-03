-- payment_option_uuid was write-only (no query ever read it back — the only
-- consumers, FindByPaymentOptionUUID/FindCompletedByPaymentOptionUUID, had no
-- callers) and CreatePayment no longer accepts a distinct value for it, so it
-- always duplicated `provider`. Drop it.
DROP INDEX IF EXISTS index_payment_option_uuid;
ALTER TABLE payments DROP COLUMN payment_option_uuid;
