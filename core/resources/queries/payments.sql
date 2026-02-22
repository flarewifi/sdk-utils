-- name: CreatePayment :one
INSERT INTO payments (uuid, purchase_id, amount, payment_option_uuid, provider)
VALUES
  (@uuid, @purchase_id, @amount, @payment_option_uuid, @provider) RETURNING id;


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


-- name: FindPurchasesByPaymentOptionUUID :many
SELECT
  purchases.*
FROM
  purchases
INNER JOIN
  payments ON payments.purchase_id = purchases.id
WHERE
  payments.payment_option_uuid = @payment_option_uuid;


-- name: FindCompletedPurchasesByPaymentOptionUUID :many
SELECT
  purchases.*
FROM
  purchases
INNER JOIN
  payments ON payments.purchase_id = purchases.id
WHERE
  payments.payment_option_uuid = @payment_option_uuid
  AND purchases.confirmed_at IS NOT NULL;

