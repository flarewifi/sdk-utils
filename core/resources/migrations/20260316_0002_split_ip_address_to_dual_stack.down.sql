-- Revert dual-stack ip columns back to a single ip_address VARCHAR(45).
-- IPv4 is preferred; if only IPv6 is set, that is used instead.

CREATE TABLE devices_new (
    id         INTEGER PRIMARY KEY,
    uuid       VARCHAR(36) NOT NULL DEFAULT '',
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    hostname   VARCHAR(64) NOT NULL DEFAULT '',
    status     INTEGER     NOT NULL DEFAULT 2,
    created_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO devices_new (id, uuid, ip_address, hostname, status, created_at, updated_at)
SELECT
    id,
    uuid,
    CASE
        WHEN ipv4_addr != '' THEN ipv4_addr
        ELSE ipv6_addr
    END AS ip_address,
    hostname,
    status,
    created_at,
    updated_at
FROM devices;

DROP TABLE devices;

ALTER TABLE devices_new RENAME TO devices;

CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid) WHERE uuid != '';
