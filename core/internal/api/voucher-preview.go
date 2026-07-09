package api

import (
	"time"

	sdkapi "sdk/api"
)

// previewVoucherBatch is an in-memory IVoucherBatch built from CreateVouchersParams
// before any DB writes. It is passed to EventVoucherBeforeCreate batch callbacks.
// ID() == 0 since the batch is not yet persisted.
type previewVoucherBatch struct {
	params      sdkapi.CreateVouchersParams
	providerPkg string
	now         time.Time
}

func (b *previewVoucherBatch) ID() int64           { return 0 }
func (b *previewVoucherBatch) UUID() string        { return b.params.BatchUUID }
func (b *previewVoucherBatch) Amount() *float64    { return b.params.Amount }
func (b *previewVoucherBatch) Metadata() string    { return "" }
func (b *previewVoucherBatch) ProviderPkg() string { return b.providerPkg }
func (b *previewVoucherBatch) VouchersCount() int64 { return int64(len(b.params.Entries)) }
func (b *previewVoucherBatch) CreatedAt() time.Time { return b.now }
func (b *previewVoucherBatch) UpdatedAt() time.Time { return b.now }

// previewVoucher is an in-memory IVoucher built from a single CreateVouchersParams
// entry plus a pre-generated code and UUID, passed to EventVoucherBeforeCreate
// voucher callbacks before the voucher INSERT. ID() == 0, Session() and Device()
// are nil.
type previewVoucher struct {
	entry       sdkapi.VoucherEntry
	batchUUID   string
	code        string
	uuid        string
	providerPkg string
	now         time.Time
}

func (v *previewVoucher) ID() int64                      { return 0 }
func (v *previewVoucher) UUID() string                   { return v.uuid }
func (v *previewVoucher) BatchUUID() string              { return v.batchUUID }
func (v *previewVoucher) Code() string                   { return v.code }
func (v *previewVoucher) ProviderPkg() string            { return v.providerPkg }
func (v *previewVoucher) Type() sdkapi.SessionType       { return v.entry.Type }
func (v *previewVoucher) TimeSecs() int64                { return v.entry.TimeSecs }
func (v *previewVoucher) DataMb() int64                  { return v.entry.DataMb }
func (v *previewVoucher) DownSpeedMbps() int64           { return v.entry.DownSpeedMbps }
func (v *previewVoucher) UpSpeedMbps() int64             { return v.entry.UpSpeedMbps }
func (v *previewVoucher) SessionExpDays() *int           { return v.entry.SessionExpDays }
func (v *previewVoucher) UseGlobal() bool                { return v.entry.UseGlobal }
func (v *previewVoucher) Session() sdkapi.IClientSession { return nil }
func (v *previewVoucher) Device() sdkapi.IClientDevice   { return nil }
func (v *previewVoucher) ExpiresAt() *time.Time          { return v.entry.ExpiresAt }
func (v *previewVoucher) ActivatedAt() *time.Time        { return nil }
func (v *previewVoucher) CreatedAt() time.Time           { return v.now }
