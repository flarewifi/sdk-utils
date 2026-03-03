-- Create a view for easy access to current MAC addresses
CREATE VIEW IF NOT EXISTS device_current_macs AS
SELECT 
    device_id,
    mac_address,
    last_seen_at
FROM device_macs
WHERE is_current = TRUE;
