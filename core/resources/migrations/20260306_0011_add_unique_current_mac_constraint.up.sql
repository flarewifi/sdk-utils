-- Add unique constraint on MAC address where is_current = TRUE
-- This ensures only ONE device can have any given MAC address as their current MAC.
-- MAC history (is_current = FALSE) can still contain the same MAC for multiple devices.

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_macs_unique_current_mac
ON device_macs(mac_address) WHERE is_current = TRUE;
