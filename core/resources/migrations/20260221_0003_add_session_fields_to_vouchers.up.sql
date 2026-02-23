-- Add session_exp_days and use_global columns to vouchers table
ALTER TABLE vouchers ADD COLUMN session_exp_days INTEGER;
ALTER TABLE vouchers ADD COLUMN use_global INTEGER NOT NULL DEFAULT 0;
