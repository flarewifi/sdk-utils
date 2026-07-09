/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"
)

type IVoucherBatch interface {
	ID() int64
	UUID() string
	Amount() *float64
	Metadata() string    // JSON metadata string
	ProviderPkg() string // Plugin package that created this batch
	// VouchersCount returns how many vouchers the batch holds. In an
	// EventVoucherBatchBeforeCreate hook the batch is a preview (not yet
	// persisted), so this is the intended count before any DB write; for a
	// persisted batch it counts the rows. This scalar has no plugin-queryable
	// equivalent at before-create time, so it stays on the API.
	VouchersCount() int64
	CreatedAt() time.Time
	UpdatedAt() time.Time
	// NOTE: To LIST the vouchers in a batch, query the core `vouchers` table
	// directly (WHERE batch_uuid = UUID()) with your plugin's own sqlc queries.
	// See the Core Database Tables guide.
}

// IVoucher represents a single voucher record.
type IVoucher interface {
	ID() int64
	UUID() string
	BatchUUID() string // returns the batch UUID that groups vouchers created together
	Code() string
	ProviderPkg() string // plugin that generated the voucher
	Type() SessionType
	TimeSecs() int64
	DataMb() int64
	DownSpeedMbps() int64
	UpSpeedMbps() int64
	SessionExpDays() *int    // nil means session never expires
	UseGlobal() bool         // true = use global bandwidth, false = use per-user bandwidth
	Session() IClientSession // returns associated session (only for activated vouchers)
	Device() IClientDevice   // returns associated device (only for activated vouchers)
	ExpiresAt() *time.Time   // nil means voucher never expires
	ActivatedAt() *time.Time // nil if not yet activated
	CreatedAt() time.Time
}

// VoucherEntry holds the per-voucher fields for CreateVouchers, so each
// voucher in a batch can carry its own type/time/data/speed/expiry/code
// instead of every voucher in the call sharing one spec.
type VoucherEntry struct {
	Type           SessionType
	TimeSecs       int64
	DataMb         int64
	DownSpeedMbps  int64      // default 10 Mbps if 0
	UpSpeedMbps    int64      // default 10 Mbps if 0
	SessionExpDays *int       // nil means session never expires
	UseGlobal      bool       // true = use global bandwidth, false = use per-user bandwidth (default: false)
	ExpiresAt      *time.Time // nil means voucher never expires
	Code           string     // optional - if empty, a code is auto-generated
}

// CreateVouchersParams holds parameters for creating a batch of vouchers.
type CreateVouchersParams struct {
	Entries []VoucherEntry
	// BatchUUID is optional - if empty, a new UUID is generated for a fresh
	// batch. If set to a batch UUID that already exists, the new vouchers are
	// appended to that existing batch instead of erroring; this lets multiple
	// CreateVouchers calls (e.g. chunked imports) grow one batch over time
	// instead of each spawning its own.
	BatchUUID string
	Amount    *float64 // optional amount for the voucher batch
}

// UpdateVoucherParams holds parameters for updating a voucher.
type UpdateVoucherParams struct {
	ID             int64
	Type           SessionType
	TimeSecs       int64
	DataMb         int64
	DownSpeedMbps  int64
	UpSpeedMbps    int64
	SessionExpDays *int       // nil means session never expires
	UseGlobal      bool       // true = use global bandwidth, false = use per-user bandwidth
	ExpiresAt      *time.Time // nil means voucher never expires
}

// ActivateVoucherParams holds parameters for activating a voucher.
// Session is created internally based on voucher settings.
type ActivateVoucherParams struct {
	ID     int64         // Voucher ID
	Device IClientDevice // Device to associate with the voucher and session
}

// VoucherActivateResult holds the result of activating a voucher.
type VoucherActivateResult struct {
	Voucher IVoucher       // The activated voucher
	Session IClientSession // The session created from the voucher
}

// UpdateVoucherBatchParams holds parameters for updating a voucher batch.
type UpdateVoucherBatchParams struct {
	UUID     string   // batch UUID to update
	Amount   *float64 // new amount (nil to clear)
	Metadata string   // new metadata (empty string to clear)
}

// IVouchersApi manages voucher lifecycle including creation, activation, and deletion.
// Each plugin gets its own scoped instance — vouchers are filtered by the plugin's package name.
type IVouchersApi interface {
	// CreateVouchers creates a batch of vouchers, one per entry in params.Entries,
	// and returns them. Each entry carries its own type/time/data/speed/expiry/code,
	// so a single call can generate a batch of mixed voucher specs. Emits
	// EventVoucherBatchCreated with the created voucher batch.
	CreateVouchers(ctx context.Context, params CreateVouchersParams) ([]IVoucher, error)

	// FindByCode finds an available (unactivated) voucher by its code.
	FindByCode(ctx context.Context, code string) (IVoucher, error)

	// FindByID finds a voucher by its database ID.
	FindByID(ctx context.Context, id int64) (IVoucher, error)

	// NOTE: There is no voucher-list/count method on this API. To list, search,
	// paginate, or count vouchers (including "available" filtering), query the core
	// `vouchers` table directly with your plugin's own sqlc queries. See the Core
	// Database Tables guide.

	// Update changes a voucher's session type, time, data, and speed settings.
	// Emits EventVoucherUpdated with the updated voucher.
	Update(ctx context.Context, params UpdateVoucherParams) (IVoucher, error)

	// Activate marks a voucher as used, creates a session based on voucher settings,
	// and associates it with the provided device.
	// Emits EventVoucherActivated with the voucher.
	// Returns VoucherActivateResult containing the activated voucher and created session.
	Activate(ctx context.Context, params ActivateVoucherParams) (VoucherActivateResult, error)

	// Delete removes a voucher by its ID.
	// Emits EventVoucherDeleted with the deleted voucher.
	Delete(ctx context.Context, id int64) error

	// DeleteActivated removes all activated vouchers for this plugin.
	// Emits EventVoucherDeleted for each deleted voucher.
	DeleteActivated(ctx context.Context) error

	// FindBatchByUUID finds a voucher batch by its UUID.
	FindBatchByUUID(ctx context.Context, batchUUID string) (IVoucherBatch, error)

	// FindBatchByCode finds a batch that contains a voucher with this code
	FindBatchByCode(ctx context.Context, code string) (IVoucherBatch, error)

	// UpdateBatch updates a voucher batch's amount and metadata.
	UpdateBatch(ctx context.Context, params UpdateVoucherBatchParams) (IVoucherBatch, error)

	// NOTE: There is no batch-list/count method on this API. To list, search,
	// paginate, or count voucher batches, query the core `voucher_batches` table
	// directly with your plugin's own sqlc queries. See the Core Database Tables guide.

	// DeleteBatch removes a voucher batch and all its vouchers by UUID.
	// Emits EventVoucherBatchDeleted with the deleted batch.
	DeleteBatch(ctx context.Context, batchUUID string) error
}
