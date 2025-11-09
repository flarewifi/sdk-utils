-- name: CreatePayment :one
INSERT INTO payments (purchase_id, amount, payment_method)
VALUES
  (@purchase_id, @amount, @payment_method) RETURNING id;


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

