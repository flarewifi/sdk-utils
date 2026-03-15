-- Replace the single ip_address column with separate ipv4_addr and ipv6_addr columns.
-- SQLite does not support ALTER COLUMN / DROP COLUMN in older versions, so we recreate
-- the table.  Both columns default to '' (empty string) so existing rows are preserved.
-- Application-layer validation ensures at least one is non-empty on write.
-- NOTE: The migration runner wraps each file in its own transaction - no BEGIN/COMMIT needed here.

CREATE TABLE devices_new (
    id         INTEGER PRIMARY KEY,
    uuid       VARCHAR(36)  NOT NULL DEFAULT '',
    ipv4_addr  VARCHAR(15)  NOT NULL DEFAULT '',
    ipv6_addr  VARCHAR(45)  NOT NULL DEFAULT '',
    hostname   VARCHAR(64)  NOT NULL DEFAULT '',
    status     INTEGER      NOT NULL DEFAULT 2,
    created_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- Migrate existing data: classify each stored ip_address as IPv4 or IPv6.
-- IPv6 addresses always contain at least one colon - IPv4 addresses do not.
INSERT INTO devices_new (id, uuid, ipv4_addr, ipv6_addr, hostname, status, created_at, updated_at)
SELECT
    id,
    uuid,
    CASE WHEN ip_address NOT LIKE '%:%' THEN ip_address ELSE '' END AS ipv4_addr,
    CASE WHEN ip_address     LIKE '%:%' THEN ip_address ELSE '' END AS ipv6_addr,
    hostname,
    status,
    created_at,
    updated_at
FROM devices;

DROP TABLE devices;

ALTER TABLE devices_new RENAME TO devices;

CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid)      WHERE uuid      != '';
CREATE        INDEX IF NOT EXISTS index_device_ipv4 ON devices(ipv4_addr) WHERE ipv4_addr != '';
CREATE        INDEX IF NOT EXISTS index_device_ipv6 ON devices(ipv6_addr) WHERE ipv6_addr != '';
