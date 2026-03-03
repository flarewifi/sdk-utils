-- Create the new MAC history table
CREATE TABLE IF NOT EXISTS device_macs (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL,
    mac_address VARCHAR(17) NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_macs_device_id ON device_macs(device_id);
CREATE INDEX IF NOT EXISTS idx_device_macs_mac_address ON device_macs(mac_address);
CREATE INDEX IF NOT EXISTS idx_device_macs_is_current ON device_macs(is_current);
CREATE INDEX IF NOT EXISTS idx_device_macs_device_mac ON device_macs(device_id, mac_address);
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_macs_unique ON device_macs(device_id, mac_address);

-- Backfill from existing devices table
INSERT INTO device_macs (device_id, mac_address, is_current, first_seen_at, last_seen_at)
SELECT 
    id,
    mac_address,
    TRUE,
    created_at,
    updated_at
FROM devices
WHERE mac_address != '';
