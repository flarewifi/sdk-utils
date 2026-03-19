-- Expand ip_address column from VARCHAR(15) to VARCHAR(45) to support IPv6.
-- IPv6 maximum length: 39 chars (full notation) + zone ID allowance = 45 chars.
-- SQLite does not support ALTER COLUMN, so we recreate the table.

CREATE TABLE devices_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
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
