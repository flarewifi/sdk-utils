-- name: CreateWallet :one
INSERT INTO wallets (device_id, balance) 
VALUES 
  ($1, $2) RETURNING id;


-- name: FindWallet :one
SELECT 
  id, 
  device_id, 
  balance, 
  created_at 
FROM 
  wallets 
WHERE 
  id = $1 
LIMIT 
  1;


-- name: UpdateWallet :exec
UPDATE 
  wallets 
SET 
  balance = $1 
WHERE 
  id = $2;


-- name: FindWalletByDeviceId :one
SELECT 
  id, 
  device_id, 
  balance, 
  created_at 
FROM 
  wallets 
WHERE 
  device_id = $1 
LIMIT 
  1;

