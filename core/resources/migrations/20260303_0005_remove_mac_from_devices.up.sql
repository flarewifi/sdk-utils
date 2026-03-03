-- SQLite doesn't support DROP COLUMN directly
-- We need to recreate the table

-- Step 1: Create new devices table without mac_address
CREATE TABLE devices_new (
    id INTEGER PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    ip_address VARCHAR(15) NOT NULL DEFAULT '',
    hostname VARCHAR(64) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy data
INSERT INTO devices_new (id, uuid, ip_address, hostname, status, created_at, updated_at)
SELECT id, uuid, ip_address, hostname, status, created_at, updated_at
FROM devices;

-- Step 3: Drop old table
DROP TABLE devices;

-- Step 4: Rename new table
ALTER TABLE devices_new RENAME TO devices;

-- Step 5: Recreate indexes (but NOT the mac_address unique index)
CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid) WHERE uuid != '';
