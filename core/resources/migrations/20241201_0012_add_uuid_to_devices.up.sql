ALTER TABLE devices ADD COLUMN uuid VARCHAR(36) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS index_device_uuid ON devices(uuid) WHERE uuid != '';
