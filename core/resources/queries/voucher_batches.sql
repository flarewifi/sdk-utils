-- name: CreateVoucherBatch :one
INSERT INTO voucher_batches (uuid, amount, metadata, provider_pkg)
VALUES (@uuid, sqlc.narg('amount'), @metadata, @provider_pkg)
RETURNING *;

-- name: FindVoucherBatchByUUID :one
SELECT *
FROM voucher_batches
WHERE uuid = @uuid
LIMIT 1;

-- name: UpdateVoucherBatch :exec
UPDATE voucher_batches
SET amount = sqlc.narg('amount'),
    metadata = @metadata,
    updated_at = CURRENT_TIMESTAMP
WHERE uuid = @uuid;

-- name: GetAllVoucherBatches :many
SELECT *
FROM voucher_batches
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetAllVoucherBatchesCount :one
SELECT COUNT(id)
FROM voucher_batches;

-- name: DeleteVoucherBatchByUUID :exec
DELETE FROM voucher_batches
WHERE uuid = @uuid;

-- name: FindVoucherBatchByCode :one
SELECT vb.*
FROM voucher_batches vb
JOIN vouchers v ON v.batch_uuid = vb.uuid
WHERE v.code = @code
LIMIT 1;
