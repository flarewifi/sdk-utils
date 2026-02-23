package api

import (
	"time"

	coreQueries "core/db/queries"
	sdkapi "sdk/api"
)

// voucherImpl wraps a coreQueries.Voucher row and implements sdkapi.IVoucher.
type voucherImpl struct {
	row       coreQueries.Voucher
	device    sdkapi.IClientDevice
	session   sdkapi.IClientSession
	batchUUID string
}

func (v *voucherImpl) ID() int64                      { return v.row.ID }
func (v *voucherImpl) UUID() string                   { return v.row.Uuid }
func (v *voucherImpl) Code() string                   { return v.row.Code }
func (v *voucherImpl) ProviderPkg() string            { return v.row.ProviderPkg }
func (v *voucherImpl) Type() sdkapi.SessionType       { return sdkapi.SessionType(v.row.SessionType) }
func (v *voucherImpl) TimeSecs() int64                { return v.row.TimeSecs }
func (v *voucherImpl) DataMb() int64                  { return v.row.DataMb }
func (v *voucherImpl) DownSpeedMbps() int64           { return v.row.DownSpeedMbps }
func (v *voucherImpl) UpSpeedMbps() int64             { return v.row.UpSpeedMbps }
func (v *voucherImpl) Device() sdkapi.IClientDevice   { return v.device }
func (v *voucherImpl) Session() sdkapi.IClientSession { return v.session }
func (v *voucherImpl) SessionExpDays() *int {
	if v.row.SessionExpDays.Valid {
		days := int(v.row.SessionExpDays.Int64)
		return &days
	}
	return nil
}
func (v *voucherImpl) UseGlobal() bool { return v.row.UseGlobal != 0 }
func (v *voucherImpl) ExpiresAt() *time.Time {
	if v.row.ExpiresAt.Valid {
		return &v.row.ExpiresAt.Time
	}
	return nil
}
func (v *voucherImpl) ActivatedAt() *time.Time {
	if v.row.ActivatedAt.Valid {
		return &v.row.ActivatedAt.Time
	}
	return nil
}
func (v *voucherImpl) CreatedAt() time.Time {
	if v.row.CreatedAt.Valid {
		return v.row.CreatedAt.Time
	}
	return time.Time{}
}
func (v *voucherImpl) BatchUUID() string { return v.batchUUID }

func wrapVoucher(row coreQueries.Voucher) sdkapi.IVoucher {
	batchUUID := ""
	if row.BatchUuid.Valid {
		batchUUID = row.BatchUuid.String
	}
	return &voucherImpl{row: row, batchUUID: batchUUID}
}
