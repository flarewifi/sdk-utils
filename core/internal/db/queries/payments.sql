-- name: CreatePayment :one
INSERT INTO payments (purchase_id, amount, optname) 
VALUES 
  ($1, $2, $3) RETURNING id;


-- name: FindPayment :one
SELECT 
  id, 
  purchase_id, 
  amount, 
  optname, 
  created_at 
FROM 
  payments 
WHERE 
  id = $1 
LIMIT 
  1;


-- name: FindAllPaymentsByPurchaseId :many
SELECT 
  id, 
  purchase_id, 
  amount, 
  optname, 
  created_at 
FROM 
  payments 
WHERE 
  purchase_id = $1;


-- name: UpdatePayment :exec
UPDATE 
  payments 
SET 
  amount = $1 
WHERE 
  id = $2;

