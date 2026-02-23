-- Revert: rename provider back to provider_pkg and add provider_name
ALTER TABLE payments RENAME COLUMN provider TO provider_pkg;
ALTER TABLE payments ADD COLUMN provider_name VARCHAR(255) NOT NULL DEFAULT '';
