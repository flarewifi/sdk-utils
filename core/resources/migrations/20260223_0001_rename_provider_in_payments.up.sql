-- Rename provider_pkg to provider and remove provider_name
ALTER TABLE payments RENAME COLUMN provider_pkg TO provider;
ALTER TABLE payments DROP COLUMN provider_name;
