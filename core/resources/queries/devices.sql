-- name: CreateDevice :one
INSERT INTO devices (
  ip_address, hostname, uuid
)
VALUES
  (
    @ip_address,
    @hostname,
    @uuid
  ) RETURNING id;


-- name: FindDevice :one
SELECT
  d.id,
  d.ip_address,
  d.hostname,
  d.uuid,
  d.created_at,
  d.updated_at,
  d.status,
  COALESCE(dm.mac_address, '') as mac_address
FROM
  devices d
  LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
WHERE
  d.id = @id
LIMIT
  1;


-- name: FindDeviceByUUID :one
SELECT
  d.id,
  d.hostname,
  d.ip_address,
  d.uuid,
  d.created_at,
  d.updated_at,
  d.status,
  COALESCE(dm.mac_address, '') as mac_address
FROM
  devices d
  LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
WHERE
  d.uuid = @uuid
LIMIT
  1;


-- name: FindDeviceByIp :one
SELECT
  d.id,
  d.hostname,
  d.ip_address,
  d.uuid,
  d.created_at,
  d.updated_at,
  d.status,
  COALESCE(dm.mac_address, '') as mac_address
FROM
  devices d
  LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
WHERE
  d.ip_address = @ip_address
LIMIT
  1;


-- name: UpdateDevice :exec
UPDATE
  devices
SET
  hostname = @hostname,
  ip_address = @ip_address,
  uuid = @uuid,
  status = @status,
  updated_at = CURRENT_TIMESTAMP
WHERE
  id = @id;


-- name: FindDevicesWithEmptyUUID :many
SELECT
  d.id,
  d.hostname,
  d.ip_address,
  COALESCE(dm.mac_address, '') as mac_address,
  d.uuid,
  d.created_at,
  d.updated_at,
  d.status
FROM
  devices d
  LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
WHERE
  d.uuid = '';


-- name: UpdateDeviceUUID :exec
UPDATE
  devices
SET
  uuid = @uuid
WHERE
  id = @id;


-- name: ResetAllDeviceStatuses :exec
UPDATE
  devices
SET
  status = 2
WHERE
  status != 2;


-- name: TransferSessionsToDevice :exec
UPDATE
  sessions
SET
  device_id = @target_device_id
WHERE
  device_id = @source_device_id;


-- name: TransferPurchasesToDevice :exec
UPDATE
  purchases
SET
  device_id = @target_device_id
WHERE
  device_id = @source_device_id;


-- name: TransferFingerprintsToDevice :exec
UPDATE
  device_fingerprints
SET
  device_id = @target_device_id
WHERE
  device_id = @source_device_id;


-- name: DeleteDevice :exec
DELETE FROM
  devices
WHERE
  id = @id;
