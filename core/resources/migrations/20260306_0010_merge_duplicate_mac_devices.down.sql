-- Down migration: Cannot undo device merges
-- Merged data (sessions, purchases, fingerprints, wallet transactions) 
-- has been consolidated and the source devices have been deleted.
-- This is an intentional data cleanup migration.

-- No-op: SELECT 1;
