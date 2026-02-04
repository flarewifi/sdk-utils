-- name: CreateDevice :one
INSERT INTO devices (
  mac_address, ip_address, hostname, uuid
)
VALUES
  (
    @mac_address,
    @ip_address,
    @hostname,
    @uuid
  ) RETURNING id;


-- name: FindDevice :one
SELECT
  id,
  mac_address,
  ip_address,
  hostname,
  uuid,
  created_at,
  status
FROM
  devices
WHERE
  id = @id
LIMIT
  1;


-- name: FindDeviceByMac :one
SELECT
  id,
  hostname,
  ip_address,
  mac_address,
  uuid,
  created_at,
  status
FROM
  devices
WHERE
  mac_address = @mac_address
LIMIT
  1;


-- name: FindDeviceByUUID :one
SELECT
  id,
  hostname,
  ip_address,
  mac_address,
  uuid,
  created_at,
  status
FROM
  devices
WHERE
  uuid = @uuid
LIMIT
  1;


-- name: UpdateDevice :exec
UPDATE
  devices
SET
  hostname = @hostname,
  ip_address = @ip_address,
  mac_address = @mac_address,
  uuid = @uuid,
  status = @status
WHERE
  id = @id;


-- name: FindDevicesWithEmptyUUID :many
SELECT
  id,
  hostname,
  ip_address,
  mac_address,
  uuid,
  created_at,
  status
FROM
  devices
WHERE
  uuid = '';


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
