-- name: CreateDevice :one
INSERT INTO devices (
  mac_address, ip_address, hostname
)
VALUES
  (
    @mac_address,
    @ip_address,
    @hostname
  ) RETURNING id;


-- name: FindDevice :one
SELECT
  id,
  mac_address,
  ip_address,
  hostname,
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
  created_at,
  status
FROM
  devices
WHERE
  mac_address = @mac_address
LIMIT
  1;


-- name: UpdateDevice :exec
UPDATE
  devices
SET
  hostname = @hostname,
  ip_address = @ip_address,
  mac_address = @mac_address,
  status = @status
WHERE
  id = @id;
