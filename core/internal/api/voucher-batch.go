package api

import (
	"context"
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
func (b *voucherBatchImpl) Vouchers(page int, perPage int, search string) ([]sdkapi.IVoucher, error) {
	ctx := context.Background()
	return b.vouchersApi.GetVouchersByBatchUUID(ctx, b.row.Uuid, page, perPage)
}
func (b *voucherBatchImpl) VouchersCount() int64 {
	ctx := context.Background()
	count, _ := b.vouchersApi.GetVouchersByBatchUUIDCount(ctx, b.row.Uuid)
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

func (self *VouchersApi) wrapManyBatches(rows []coreQueries.VoucherBatch) []sdkapi.IVoucherBatch {
	result := make([]sdkapi.IVoucherBatch, len(rows))
	for i, row := range rows {
		result[i] = self.wrapBatch(row)
	}
	return result
}
