-- Revert ip_address column back to VARCHAR(15) (IPv4 only).
-- WARNING: Any IPv6 addresses stored will be truncated to 15 characters.

CREATE TABLE devices_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    ip_address VARCHAR(15) NOT NULL DEFAULT '',
    hostname VARCHAR(64) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO devices_new (id, uuid, ip_address, hostname, status, created_at, updated_at)
SELECT id, uuid, ip_address, hostname, status, created_at, updated_at
FROM devices;

DROP TABLE devices;

ALTER TABLE devices_new RENAME TO devices;

CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid) WHERE uuid != '';
