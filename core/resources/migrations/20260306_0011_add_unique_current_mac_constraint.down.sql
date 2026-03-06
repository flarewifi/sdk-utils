-- Remove the unique constraint on current MAC addresses
DROP INDEX IF EXISTS idx_device_macs_unique_current_mac;
