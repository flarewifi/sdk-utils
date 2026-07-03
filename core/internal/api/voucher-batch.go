package api

import (
	"context"
	"database/sql"
	"time"

	coreQueries "core/db/queries"
	sdkapi "sdk/api"
)

// voucherBatchImpl wraps a coreQueries.VoucherBatch row and implements sdkapi.IVoucherBatch.
type voucherBatchImpl struct {
	row         coreQueries.VoucherBatch
	vouchersApi *VouchersApi
}

func (b *voucherBatchImpl) ID() int64    { return b.row.ID }
func (b *voucherBatchImpl) UUID() string { return b.row.Uuid }
func (b *voucherBatchImpl) Amount() *float64 {
	if !b.row.Amount.Valid {
		return nil
	}
	return &b.row.Amount.Float64
}
func (b *voucherBatchImpl) Metadata() string {
	if b.row.Metadata.Valid {
		return b.row.Metadata.String
	}
	return ""
}
func (b *voucherBatchImpl) ProviderPkg() string {
	return b.row.ProviderPkg
}
func (b *voucherBatchImpl) VouchersCount() int64 {
	q := coreQueries.New(b.vouchersApi.pluginApi.db.DB)
	count, _ := q.GetVouchersByBatchUUIDCount(context.Background(), sql.NullString{String: b.row.Uuid, Valid: true})
	return count
}
func (b *voucherBatchImpl) CreatedAt() time.Time {
	if b.row.CreatedAt.Valid {
		return b.row.CreatedAt.Time
	}
	return time.Time{}
}
func (b *voucherBatchImpl) UpdatedAt() time.Time {
	if b.row.UpdatedAt.Valid {
		return b.row.UpdatedAt.Time
	}
	return time.Time{}
}

func (self *VouchersApi) wrapBatch(row coreQueries.VoucherBatch) sdkapi.IVoucherBatch {
	return &voucherBatchImpl{row: row, vouchersApi: self}
}
