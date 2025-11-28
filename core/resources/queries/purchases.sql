-- name: CreatePurchase :one
INSERT INTO purchases (uid, device_id, sku, name, description, price, any_price, callback_plugin, callback_route, webhook_route, metadata, processing, payment_url)
    VALUES (@uid, @device_id, @sku, @name, @description, @price, @any_price, @callback_plugin, @callback_route, @webhook_route, @metadata, @processing, @payment_url)
RETURNING
    id;

-- name: FindPurchase :one
SELECT
    *
FROM
    purchases
WHERE
    id = @id
LIMIT 1;

-- name: FindPurchaseByUID :one
SELECT
    *
FROM
    purchases
WHERE
    uid = @uid
LIMIT 1;

-- name: FindPurchaseByDeviceId :one
SELECT
    *
FROM
    purchases
WHERE
    device_id = @id
LIMIT 1;

-- name: UpdatePurchase :exec
UPDATE
    purchases
SET
    wallet_debit = @wallet_debit,
    wallet_tx_id = @wallet_tx_id,
    cancelled_at = @cancelled_at,
    confirmed_at = @confirmed_at,
    cancelled_reason = @cancelled_reason,
    processing = @processing,
    payment_url = @payment_url
WHERE
    id = @id;

-- name: FindPendingPurchase :one
SELECT
    *
FROM
    purchases
WHERE
    confirmed_at IS NULL
    AND cancelled_at IS NULL
    AND device_id = @device_id
LIMIT 1;

-- name: UpdatePurchaseMetadata :exec
UPDATE
    purchases
SET
    metadata = @metadata
WHERE
    id = @id;

