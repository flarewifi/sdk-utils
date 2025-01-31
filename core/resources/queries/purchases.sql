-- name: CreatePurchase :one
INSERT INTO purchases (device_id, sku, name, description, price, any_price, callback_plugin, callback_route, callback_route_params)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING
    id;

-- name: FindPurchase :one
SELECT
    *
FROM
    purchases
WHERE
    id = $1
LIMIT 1;

-- name: FindPurchaseByDeviceId :one
SELECT
    *
FROM
    purchases
WHERE
    device_id = $1
LIMIT 1;

-- name: UpdatePurchase :exec
UPDATE
    purchases
SET
    wallet_debit = $1,
    wallet_tx_id = $2,
    cancelled_at = $3,
    confirmed_at = $4,
    cancelled_reason = $5
WHERE
    id = $6;

-- name: FindPending :one
SELECT
    *
FROM
    purchases
WHERE
    confirmed_at IS NULL
    AND cancelled_at IS NULL
    AND device_id = $1
LIMIT 1;

