-- name: CreateWalletTrns :one
INSERT INTO wallet_transactions (
  wallet_id, amount, new_balance, description
) 
VALUES 
  (@wallet_id, @amount, @new_balance, @description)
RETURNING *;


-- name: FindWalletTrns :one
SELECT 
  id, 
  wallet_id, 
  amount, 
  new_balance, 
  description, 
  created_at 
FROM 
  wallet_transactions 
WHERE 
  id = @id 
LIMIT 
  1;


-- name: UpdateWalletTrns :exec
UPDATE
  wallet_transactions 
SET 
  wallet_id = @wallet_id,
  amount = @amount,
  new_balance = @new_balance,
  description = @description 
WHERE 
  id = @id;
