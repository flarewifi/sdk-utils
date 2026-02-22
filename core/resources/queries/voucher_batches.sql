-- name: CreateVoucherBatch :one
INSERT INTO voucher_batches (
    uuid,
    total_amount,
    payment_note
) VALUES (
    @uuid,
    @total_amount,
    @payment_note
)
RETURNING *;

-- name: FindVoucherBatchByUUID :one
SELECT * FROM voucher_batches
WHERE uuid = @uuid
LIMIT 1;
