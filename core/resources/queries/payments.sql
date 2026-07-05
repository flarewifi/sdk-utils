-- name: CreatePayment :one
INSERT INTO payments (uuid, purchase_id, amount, provider, payment_method)
VALUES
  (@uuid, @purchase_id, @amount, @provider, @payment_method) RETURNING id;


-- name: FindPayment :one
SELECT *
FROM
  payments
WHERE
  id = @id
LIMIT
  1;


-- name: FindAllPaymentsByPurchaseId :many
SELECT *
FROM
  payments
WHERE
  purchase_id = @purchase_id;


-- name: UpdatePayment :exec
UPDATE
  payments
SET
  amount = @amount
WHERE
  id = @id;

