-- Add UNIQUE constraint to voucher codes (globally unique across all providers)
CREATE UNIQUE INDEX idx_vouchers_code_unique ON vouchers(code);
