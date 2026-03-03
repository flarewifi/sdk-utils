-- Restore mac_address column
CREATE TABLE devices_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    ip_address VARCHAR(15) NOT NULL DEFAULT '',
    mac_address VARCHAR(17) NOT NULL DEFAULT '',
    hostname VARCHAR(64) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Copy data back and restore MAC from client_device_macs
INSERT INTO devices_new (id, uuid, ip_address, mac_address, hostname, status, created_at, updated_at)
SELECT 
    d.id,
    d.uuid,
    d.ip_address,
    COALESCE(cdm.mac_address, ''),
    d.hostname,
    d.status,
    d.created_at,
    d.updated_at
FROM devices d
LEFT JOIN client_device_macs cdm ON d.id = cdm.device_id AND cdm.is_current = TRUE;

DROP TABLE devices;
ALTER TABLE devices_new RENAME TO devices;

CREATE UNIQUE INDEX IF NOT EXISTS index_mac_address ON devices(mac_address);
CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid) WHERE uuid != '';
