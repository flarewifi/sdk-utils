-- name: CreateDevice :one
INSERT INTO devices (
  mac_address, ip_address, hostname
) 
VALUES 
  (
    $1, 
    $2, 
    $3
  ) RETURNING id;


-- name: FindDevice :one
SELECT 
  id, 
  mac_address, 
  ip_address, 
  hostname, 
  created_at 
FROM 
  devices 
WHERE 
  id = $1 
LIMIT 
  1;


-- name: FindDeviceByMac :one
SELECT 
  id, 
  hostname, 
  ip_address, 
  mac_address, 
  created_at 
FROM 
  devices 
WHERE 
  mac_address = $1
LIMIT 
  1;


-- name: UpdateDevice :exec
UPDATE 
  devices 
SET 
  hostname = $1, 
  ip_address = $2, 
  mac_address = $3 
WHERE 
  id = $4;

