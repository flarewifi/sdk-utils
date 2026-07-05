-- Add payment_method to track the specific method used within a payment provider
-- (e.g. "Coins", "Bills") — distinct from `provider` (which plugin processed it)
-- and `payment_option_uuid` (the old payment_method column, renamed 2026-01-24).
ALTER TABLE payments ADD COLUMN payment_method VARCHAR(255) NOT NULL DEFAULT '';
