-- Remove UNIQUE constraint from voucher codes
DROP INDEX IF EXISTS idx_vouchers_code_unique;
