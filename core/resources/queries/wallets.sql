-- name: CreateWallet :one
INSERT INTO wallets (device_id, balance) 
VALUES 
  (@device_id, @balance) RETURNING id;


-- name: FindWallet :one
SELECT 
  id, 
  device_id, 
  balance, 
  created_at 
FROM 
  wallets 
WHERE 
  id = @id 
LIMIT 
  1;


-- name: UpdateWallet :exec
UPDATE 
  wallets 
SET 
  balance = @balance 
WHERE 
  id = @id;


-- name: FindWalletByDeviceId :one
SELECT 
  id, 
  device_id, 
  balance, 
  created_at 
FROM 
  wallets 
WHERE 
  device_id = @device_id 
LIMIT 
  1;

