package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"slices"
	"sync"
	"time"

	coreQueries "core/db/queries"
	sdkapi "sdk/api"
)

// Global voucher event handlers shared across all plugin instances
var (
	globalVoucherMu            sync.Mutex
	globalVoucherHandlers      = make(map[sdkapi.VoucherEvent][]func(sdkapi.IVoucher) error)
	globalVoucherBatchHandlers = make(map[sdkapi.VoucherEvent][]func(sdkapi.IVoucherBatch) error)
	globalBeforeCreateHandlers []func(context.Context, *sdkapi.CreateVouchersParams) error
)

func NewVouchersApi(pluginApi *PluginApi) *VouchersApi {
	v := &VouchersApi{
		pluginApi: pluginApi,
	}
	pluginApi.VouchersAPI = v
	return v
}

// VouchersApi implements sdkapi.IVouchersApi, scoped per plugin package.
// Event handlers are stored globally to enable cross-plugin event delivery.
type VouchersApi struct {
	pluginApi *PluginApi
}

// wrapWithRelations wraps a voucher row and eagerly loads device/session for activated vouchers.
func (self *VouchersApi) wrapWithRelations(ctx context.Context, row coreQueries.Voucher) sdkapi.IVoucher {
	batchUUID := ""
	if row.BatchUuid.Valid {
		batchUUID = row.BatchUuid.String
	}
	v := &voucherImpl{row: row, batchUUID: batchUUID}
	if row.DeviceID.Valid {
		device, err := self.pluginApi.SessionMgr.FindDeviceByID(ctx, row.DeviceID.Int64)
		if err == nil {
			v.device = device
		}
	}
	if row.SessionID.Valid {
		session, err := self.pluginApi.SessionMgr.FindSessionByID(ctx, row.SessionID.Int64)
		if err == nil {
			v.session = session
		}
	}
	return v
}

func (self *VouchersApi) wrapManyWithRelations(ctx context.Context, rows []coreQueries.Voucher) []sdkapi.IVoucher {
	result := make([]sdkapi.IVoucher, len(rows))
	for i, row := range rows {
		result[i] = self.wrapWithRelations(ctx, row)
	}
	return result
}

func (self *VouchersApi) providerPkg() string {
	return self.pluginApi.Info().Package
}

func (self *VouchersApi) emitSingle(event sdkapi.VoucherEvent, v sdkapi.IVoucher) {
	// Use global handlers to ensure cross-plugin event delivery
	globalVoucherMu.Lock()
	callbacks := globalVoucherHandlers[event]
	callbacksCopy := make([]func(sdkapi.IVoucher) error, len(callbacks))
	copy(callbacksCopy, callbacks)
	globalVoucherMu.Unlock()

	for _, cb := range callbacksCopy {
		if err := cb(v); err != nil {
			log.Printf("[VouchersApi] Error in %s handler: %v", event, err)
		}
	}
}

func (self *VouchersApi) emitBatch(event sdkapi.VoucherEvent, batch sdkapi.IVoucherBatch) {
	// Use global handlers to ensure cross-plugin event delivery
	globalVoucherMu.Lock()
	callbacks := globalVoucherBatchHandlers[event]
	callbacksCopy := make([]func(sdkapi.IVoucherBatch) error, len(callbacks))
	copy(callbacksCopy, callbacks)
	globalVoucherMu.Unlock()

	for _, cb := range callbacksCopy {
		if err := cb(batch); err != nil {
			log.Printf("[VouchersApi] Error in %s batch handler: %v", event, err)
		}
	}
}

// CreateVouchers creates a batch of vouchers and emits EventVoucherGenerated.
// Automatically creates a voucher batch record before creating vouchers.
func (self *VouchersApi) CreateVouchers(ctx context.Context, params sdkapi.CreateVouchersParams) ([]sdkapi.IVoucher, error) {
	// Run before-create hooks (can modify params or return error to block)
	globalVoucherMu.Lock()
	handlers := make([]func(context.Context, *sdkapi.CreateVouchersParams) error, len(globalBeforeCreateHandlers))
	copy(handlers, globalBeforeCreateHandlers)
	globalVoucherMu.Unlock()

	for _, hook := range handlers {
		if err := hook(ctx, &params); err != nil {
			return nil, err
		}
	}

	db := self.pluginApi.db
	q := coreQueries.New(db.DB)

	// Apply default bandwidth if not specified
	downSpeedMbps := params.DownSpeedMbps
	upSpeedMbps := params.UpSpeedMbps
	if downSpeedMbps == 0 {
		downSpeedMbps = 10
	}
	if upSpeedMbps == 0 {
		upSpeedMbps = 10
	}

	// Use provided BatchUUID or generate one
	batchUUID := params.BatchUUID
	if batchUUID == "" {
		batchUUID = generateUUID()
		params.BatchUUID = batchUUID // Update params so hooks can access it
	}

	// Automatically create a voucher batch record
	amount := sql.NullFloat64{}
	if params.Amount != nil {
		amount = sql.NullFloat64{Float64: *params.Amount, Valid: true}
	}
	_, err := q.CreateVoucherBatch(ctx, coreQueries.CreateVoucherBatchParams{
		Uuid:        batchUUID,
		Amount:      amount,
		Metadata:    sql.NullString{},
		ProviderPkg: self.providerPkg(), // Track which plugin generated this batch
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create voucher batch: %w", err)
	}

	var created []sdkapi.IVoucher
	for i := 0; i < params.Count; i++ {
		code := generateVoucherCode()
		uuid := generateVoucherUUID()
		expiresAt := sql.NullTime{}
		if params.ExpiresAt != nil {
			expiresAt = sql.NullTime{Time: *params.ExpiresAt, Valid: true}
		}
		sessionExpDays := sql.NullInt64{}
		if params.SessionExpDays != nil {
			sessionExpDays = sql.NullInt64{Int64: int64(*params.SessionExpDays), Valid: true}
		}
		useGlobal := int64(0)
		if params.UseGlobal {
			useGlobal = 1
		}
		batchUUIDParam := sql.NullString{String: batchUUID, Valid: true}
		row, err := q.CreateVoucher(ctx, coreQueries.CreateVoucherParams{
			Uuid:           uuid,
			Code:           code,
			ProviderPkg:    self.providerPkg(),
			SessionType:    string(params.Type),
			TimeSecs:       params.TimeSecs,
			DataMb:         params.DataMb,
			DownSpeedMbps:  downSpeedMbps,
			UpSpeedMbps:    upSpeedMbps,
			SessionExpDays: sessionExpDays,
			UseGlobal:      useGlobal,
			ExpiresAt:      expiresAt,
			BatchUuid:      batchUUIDParam,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to create voucher: %w", err)
		}
		created = append(created, wrapVoucher(row))
	}

	if len(created) > 0 {
		// Fetch the batch to emit the event
		batch, err := self.FindBatchByUUID(ctx, batchUUID)
		if err == nil {
			self.emitBatch(sdkapi.EventVoucherGenerated, batch)
		}
	}

	return created, nil
}

// GetVouchersByBatchUUID returns vouchers with the given batch UUID with pagination.
func (self *VouchersApi) GetVouchersByBatchUUID(ctx context.Context, batchUUID string, page int, perPage int) ([]sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	offset := int64(perPage * (page - 1))
	batchUuidParam := sql.NullString{String: batchUUID, Valid: true}
	rows, err := q.GetVouchersByBatchUUID(ctx, coreQueries.GetVouchersByBatchUUIDParams{
		BatchUuid: batchUuidParam,
		RowLimit:  int64(perPage),
		RowOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get vouchers by batch: %w", err)
	}
	return self.wrapManyWithRelations(ctx, rows), nil
}

// GetVouchersByBatchUUIDCount returns the count of vouchers with the given batch UUID.
func (self *VouchersApi) GetVouchersByBatchUUIDCount(ctx context.Context, batchUUID string) (int64, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	batchUuidParam := sql.NullString{String: batchUUID, Valid: true}
	count, err := q.GetVouchersByBatchUUIDCount(ctx, batchUuidParam)
	if err != nil {
		return 0, fmt.Errorf("unable to count vouchers by batch: %w", err)
	}
	return count, nil
}

// FindByCode finds an available voucher by code (global search across all providers).
// The provider_pkg field is preserved for historical tracking only.
func (self *VouchersApi) FindByCode(ctx context.Context, code string) (sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	row, err := q.FindVoucherByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("voucher not found: %w", err)
	}
	return self.wrapWithRelations(ctx, row), nil
}

// FindByID finds a voucher by its database ID.
func (self *VouchersApi) FindByID(ctx context.Context, id int64) (sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	row, err := q.FindVoucherByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("voucher not found: %w", err)
	}
	return self.wrapWithRelations(ctx, row), nil
}

// List returns a paginated list of vouchers for this plugin.
func (self *VouchersApi) List(ctx context.Context, params sdkapi.ListVouchersParams) (sdkapi.ListVouchersResult, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	offset := int64(params.PerPage * (params.Page - 1))

	// Use filtered query if any filters are provided
	if params.Search != nil || params.IsActivated != nil || params.DateStart != nil || params.DateEnd != nil {
		// Prepare search parameter
		var search interface{}
		if params.Search != nil && *params.Search != "" {
			search = *params.Search
		}

		fmt.Println(params.DateStart)
		fmt.Println(params.DateEnd)

		// Prepare isActivated parameter (convert *bool to int 0/1 or nil)
		var isActivated interface{}
		var dateStart interface{}
		var dateEnd interface{}

		if params.IsActivated != nil {
			if *params.IsActivated {
				isActivated = 1
			} else {
				isActivated = 0
			}
		}

		if params.DateStart != nil && params.DateEnd != nil {
			dateStart = *params.DateStart
			dateEnd = *params.DateEnd
		}

		rows, err := q.GetVouchersFiltered(ctx, coreQueries.GetVouchersFilteredParams{
			ProviderPkg: self.providerPkg(),
			Search:      search,
			IsActivated: isActivated,
			RowLimit:    int64(params.PerPage),
			RowOffset:   offset,
			DateStart:   dateStart,
			DateEnd:     dateEnd,
		})
		if err != nil {
			return sdkapi.ListVouchersResult{}, fmt.Errorf("unable to list vouchers: %w", err)
		}

		count, err := q.GetVouchersFilteredCount(ctx, coreQueries.GetVouchersFilteredCountParams{
			ProviderPkg: self.providerPkg(),
			Search:      search,
			IsActivated: isActivated,
		})
		if err != nil {
			return sdkapi.ListVouchersResult{}, fmt.Errorf("unable to count vouchers: %w", err)
		}

		return sdkapi.ListVouchersResult{
			Vouchers: self.wrapManyWithRelations(ctx, rows),
			Count:    count,
		}, nil
	}

	// Use unfiltered query for backward compatibility
	rows, err := q.GetAllVouchers(ctx, coreQueries.GetAllVouchersParams{
		ProviderPkg: self.providerPkg(),
		RowLimit:    int64(params.PerPage),
		RowOffset:   offset,
	})
	if err != nil {
		return sdkapi.ListVouchersResult{}, fmt.Errorf("unable to list vouchers: %w", err)
	}

	count, err := q.GetAllVouchersCount(ctx, self.providerPkg())
	if err != nil {
		return sdkapi.ListVouchersResult{}, fmt.Errorf("unable to count vouchers: %w", err)
	}

	return sdkapi.ListVouchersResult{
		Vouchers: self.wrapManyWithRelations(ctx, rows),
		Count:    count,
	}, nil
}

// Count returns the total number of vouchers for this plugin.
func (self *VouchersApi) Count(ctx context.Context) (int64, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	count, err := q.GetAllVouchersCount(ctx, self.providerPkg())
	if err != nil {
		return 0, fmt.Errorf("unable to count vouchers: %w", err)
	}
	return count, nil
}

// Update changes a voucher's session type, time, data, and speed settings, and emits EventVoucherUpdated.
func (self *VouchersApi) Update(ctx context.Context, params sdkapi.UpdateVoucherParams) (sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	expiresAt := sql.NullTime{}
	if params.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *params.ExpiresAt, Valid: true}
	}
	sessionExpDays := sql.NullInt64{}
	if params.SessionExpDays != nil {
		sessionExpDays = sql.NullInt64{Int64: int64(*params.SessionExpDays), Valid: true}
	}
	useGlobal := int64(0)
	if params.UseGlobal {
		useGlobal = 1
	}
	err := q.UpdateVoucher(ctx, coreQueries.UpdateVoucherParams{
		SessionType:    string(params.Type),
		TimeSecs:       params.TimeSecs,
		DataMb:         params.DataMb,
		DownSpeedMbps:  params.DownSpeedMbps,
		UpSpeedMbps:    params.UpSpeedMbps,
		SessionExpDays: sessionExpDays,
		UseGlobal:      useGlobal,
		ExpiresAt:      expiresAt,
		ID:             params.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to update voucher: %w", err)
	}

	updated, err := self.FindByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	self.emitSingle(sdkapi.EventVoucherUpdated, updated)
	return updated, nil
}

// Activate marks a voucher as used, creates a session based on voucher settings,
// and associates it with the provided device.
// Returns VoucherActivateResult containing the activated voucher and created session.
func (self *VouchersApi) Activate(ctx context.Context, params sdkapi.ActivateVoucherParams) (sdkapi.VoucherActivateResult, error) {
	q := coreQueries.New(self.pluginApi.db.DB)

	// Find the voucher first
	voucher, err := self.FindByID(ctx, params.ID)
	if err != nil {
		return sdkapi.VoucherActivateResult{}, fmt.Errorf("unable to find voucher: %w", err)
	}

	// Reject expired vouchers
	if exp := voucher.ExpiresAt(); exp != nil && exp.Before(time.Now().UTC()) {
		return sdkapi.VoucherActivateResult{}, fmt.Errorf("voucher has expired")
	}

	// Determine bandwidth settings
	downMbits := int(voucher.DownSpeedMbps())
	upMbits := int(voucher.UpSpeedMbps())
	useGlobal := voucher.UseGlobal()

	// Apply defaults if not specified (10 Mbps)
	if downMbits == 0 {
		downMbits = 10
	}
	if upMbits == 0 {
		upMbits = 10
	}

	// Generate a UUID for the session
	sessionUUID := generateSessionUUID()

	// Create the session based on voucher settings
	session, err := self.pluginApi.SessionsMgrAPI.CreateSession(ctx, sdkapi.CreateSessionParams{
		UUID:        sessionUUID,
		DevId:       params.Device.ID(),
		SessionType: voucher.Type(),
		TimeSecs:    int(voucher.TimeSecs()),
		DataMbytes:  float64(voucher.DataMb()),
		ExpDays:     voucher.SessionExpDays(),
		DownMbits:   downMbits,
		UpMbits:     upMbits,
		UseGlobal:   useGlobal,
	})
	if err != nil {
		return sdkapi.VoucherActivateResult{}, fmt.Errorf("unable to create session from voucher: %w", err)
	}

	// Mark the voucher as activated with the created session
	err = q.ActivateVoucher(ctx, coreQueries.ActivateVoucherParams{
		SessionID: sql.NullInt64{Int64: session.ID(), Valid: true},
		DeviceID:  sql.NullInt64{Int64: params.Device.ID(), Valid: true},
		ID:        params.ID,
	})
	if err != nil {
		return sdkapi.VoucherActivateResult{}, fmt.Errorf("unable to activate voucher: %w", err)
	}

	// Fetch the updated voucher row
	row, err := q.FindVoucherByID(ctx, params.ID)
	if err != nil {
		return sdkapi.VoucherActivateResult{}, fmt.Errorf("unable to fetch activated voucher: %w", err)
	}
	activated := &voucherImpl{
		row:     row,
		device:  params.Device,
		session: session,
	}

	self.emitSingle(sdkapi.EventVoucherActivated, activated)
	return sdkapi.VoucherActivateResult{
		Voucher: activated,
		Session: session,
	}, nil
}

// generateSessionUUID generates a unique session identifier
func generateSessionUUID() string {
	return generateUUID()
}

// generateVoucherUUID generates a unique voucher identifier
func generateVoucherUUID() string {
	return generateUUID()
}

// generateUUID generates a UUID v4 format string
func generateUUID() string {
	const charset = "0123456789abcdef"
	b := make([]byte, 32)
	for i := range b {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[v.Int64()]
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", string(b[0:8]), string(b[8:12]), string(b[12:16]), string(b[16:20]), string(b[20:32]))
}

// Delete removes a voucher and emits EventVoucherDeleted.
func (self *VouchersApi) Delete(ctx context.Context, id int64) error {
	voucher, err := self.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("unable to find voucher before delete: %w", err)
	}

	q := coreQueries.New(self.pluginApi.db.DB)
	if err := q.DeleteVoucherByID(ctx, id); err != nil {
		return fmt.Errorf("unable to delete voucher: %w", err)
	}

	self.emitSingle(sdkapi.EventVoucherDeleted, voucher)
	return nil
}

// DeleteActivated removes all activated vouchers and emits EventVoucherDeleted per voucher.
func (self *VouchersApi) DeleteActivated(ctx context.Context) error {
	activated, err := self.getActivated(ctx)
	if err != nil {
		return err
	}

	q := coreQueries.New(self.pluginApi.db.DB)
	err = q.DeleteActivatedVouchers(ctx, self.providerPkg())
	if err != nil {
		return fmt.Errorf("unable to delete activated vouchers: %w", err)
	}

	for _, v := range activated {
		self.emitSingle(sdkapi.EventVoucherDeleted, v)
	}
	return nil
}

// GetAvailable returns all unactivated vouchers for this plugin.
func (self *VouchersApi) GetAvailable(ctx context.Context) ([]sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	rows, err := q.GetAvailableVouchers(ctx, self.providerPkg())
	if err != nil {
		return nil, fmt.Errorf("unable to get available vouchers: %w", err)
	}
	return self.wrapManyWithRelations(ctx, rows), nil
}

// getActivated returns all activated vouchers (internal use for DeleteActivated).
func (self *VouchersApi) getActivated(ctx context.Context) ([]sdkapi.IVoucher, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	rows, err := q.GetActivatedVouchers(ctx, self.providerPkg())
	if err != nil {
		return nil, fmt.Errorf("unable to get activated vouchers: %w", err)
	}
	return self.wrapManyWithRelations(ctx, rows), nil
}

// OnVoucherEvent registers a callback for a single-voucher event (Activated, Updated, Deleted).
// Handlers are registered globally to allow cross-plugin event delivery.
func (self *VouchersApi) OnVoucherEvent(event sdkapi.VoucherEvent, callback func(sdkapi.IVoucher) error) {
	globalVoucherMu.Lock()
	defer globalVoucherMu.Unlock()
	globalVoucherHandlers[event] = append(globalVoucherHandlers[event], callback)
}

// OnVoucherBatchEvent registers a callback for a batch voucher event (Generated).
// Handlers are registered globally to allow cross-plugin event delivery.
func (self *VouchersApi) OnVoucherBatchEvent(event sdkapi.VoucherEvent, callback func(sdkapi.IVoucherBatch) error) {
	globalVoucherMu.Lock()
	defer globalVoucherMu.Unlock()
	globalVoucherBatchHandlers[event] = append(globalVoucherBatchHandlers[event], callback)
}

// OnBeforeCreate registers a hook called before voucher creation.
// The hook receives a pointer to params and can modify them.
// Return an error to block creation.
// Handlers are registered globally to allow cross-plugin hook delivery.
func (self *VouchersApi) OnBeforeCreate(callback func(context.Context, *sdkapi.CreateVouchersParams) error) {
	globalVoucherMu.Lock()
	defer globalVoucherMu.Unlock()
	globalBeforeCreateHandlers = append(globalBeforeCreateHandlers, callback)
}

// FindBatchByUUID finds a voucher batch by its UUID.
func (self *VouchersApi) FindBatchByUUID(ctx context.Context, batchUUID string) (sdkapi.IVoucherBatch, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	row, err := q.FindVoucherBatchByUUID(ctx, batchUUID)
	if err != nil {
		return nil, fmt.Errorf("voucher batch not found: %w", err)
	}
	return self.wrapBatch(row), nil
}

// FindBatchByCode finds a voucher batch that contains a voucher with the given code.
func (self *VouchersApi) FindBatchByCode(ctx context.Context, code string) (sdkapi.IVoucherBatch, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	row, err := q.FindVoucherBatchByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("voucher batch not found for code: %w", err)
	}
	return self.wrapBatch(row), nil
}

// UpdateBatch updates a voucher batch's amount and metadata.
func (self *VouchersApi) UpdateBatch(ctx context.Context, params sdkapi.UpdateVoucherBatchParams) (sdkapi.IVoucherBatch, error) {
	q := coreQueries.New(self.pluginApi.db.DB)

	amount := sql.NullFloat64{}
	if params.Amount != nil {
		amount = sql.NullFloat64{Float64: *params.Amount, Valid: true}
	}

	metadata := sql.NullString{}
	if params.Metadata != "" {
		metadata = sql.NullString{String: params.Metadata, Valid: true}
	}

	err := q.UpdateVoucherBatch(ctx, coreQueries.UpdateVoucherBatchParams{
		Uuid:     params.UUID,
		Amount:   amount,
		Metadata: metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to update voucher batch: %w", err)
	}

	return self.FindBatchByUUID(ctx, params.UUID)
}

// ListBatches returns a paginated list of voucher batches.
func (self *VouchersApi) ListBatches(ctx context.Context, params sdkapi.ListVoucherBatchesParams) (sdkapi.ListVoucherBatchesResult, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	offset := int64(params.PerPage * (params.Page - 1))

	rows, err := q.GetAllVoucherBatches(ctx, coreQueries.GetAllVoucherBatchesParams{
		RowLimit:  int64(params.PerPage),
		RowOffset: offset,
	})
	if err != nil {
		return sdkapi.ListVoucherBatchesResult{}, fmt.Errorf("unable to list voucher batches: %w", err)
	}

	count, err := q.GetAllVoucherBatchesCount(ctx)
	if err != nil {
		return sdkapi.ListVoucherBatchesResult{}, fmt.Errorf("unable to count voucher batches: %w", err)
	}

	return sdkapi.ListVoucherBatchesResult{
		Batches: self.wrapManyBatches(rows),
		Count:   count,
	}, nil
}

// CountVouchers returns the total count of vouchers matching the filter criteria.
func (self *VouchersApi) CountVouchers(ctx context.Context, params sdkapi.ListVouchersParams) (int64, error) {
	q := coreQueries.New(self.pluginApi.db.DB)

	// Use filtered count if any filters are provided
	if params.Search != nil || params.IsActivated != nil {
		// Prepare search parameter
		var search interface{}
		if params.Search != nil && *params.Search != "" {
			search = *params.Search
		}

		// Prepare isActivated parameter (convert *bool to int 0/1 or nil)
		var isActivated interface{}
		if params.IsActivated != nil {
			if *params.IsActivated {
				isActivated = 1
			} else {
				isActivated = 0
			}
		}

		count, err := q.GetVouchersFilteredCount(ctx, coreQueries.GetVouchersFilteredCountParams{
			ProviderPkg: self.providerPkg(),
			Search:      search,
			IsActivated: isActivated,
		})
		if err != nil {
			return 0, fmt.Errorf("unable to count vouchers: %w", err)
		}
		return count, nil
	}

	// Use unfiltered count for backward compatibility
	count, err := q.GetAllVouchersCount(ctx, self.providerPkg())
	if err != nil {
		return 0, fmt.Errorf("unable to count vouchers: %w", err)
	}
	return count, nil
}

// CountBatches returns the total count of voucher batches matching the filter criteria.
func (self *VouchersApi) CountBatches(ctx context.Context, params sdkapi.ListVoucherBatchesParams) (int64, error) {
	q := coreQueries.New(self.pluginApi.db.DB)

	count, err := q.GetAllVoucherBatchesCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to count voucher batches: %w", err)
	}
	return count, nil
}

// DeleteBatch removes a voucher batch and all its vouchers by UUID.
// Emits EventVoucherBatchDeleted with the deleted batch.
func (self *VouchersApi) DeleteBatch(ctx context.Context, batchUUID string) error {
	q := coreQueries.New(self.pluginApi.db.DB)

	// Find the batch first to emit event with batch data
	batch, err := self.FindBatchByUUID(ctx, batchUUID)
	if err != nil {
		return fmt.Errorf("unable to find voucher batch: %w", err)
	}

	// Delete all vouchers in the batch first
	if err := q.DeleteVouchersByBatchUUID(ctx, sql.NullString{String: batchUUID, Valid: true}); err != nil {
		return fmt.Errorf("unable to delete vouchers in batch: %w", err)
	}

	// Delete the batch record
	if err := q.DeleteVoucherBatchByUUID(ctx, batchUUID); err != nil {
		return fmt.Errorf("unable to delete voucher batch: %w", err)
	}

	// Emit batch deleted event
	self.emitBatch(sdkapi.EventVoucherBatchDeleted, batch)

	return nil
}

// generateVoucherCode generates a random 6-character voucher code avoiding confusable characters.
func generateVoucherCode() string {
	codeLength := 6
	charset := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := make([]byte, codeLength)

	confusables := [][]byte{
		{'0', 'O', 'D', 'Q'},
		{'1', 'I', 'L'},
		{'5', 'S'},
		{'2', 'Z'},
		{'3', '8', 'B'},
		{'6', 'G', 'C'},
		{'4', 'A'},
		{'7', 'T'},
		{'U', 'V', 'Y'},
	}

	used := make(map[byte]bool)
	for i := range bytes {
		var c byte
		valid := false
		for !valid {
			v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			c = charset[v.Int64()]

			var conflict bool
			for _, group := range confusables {
				usedGroup := false
				for _, char := range group {
					if used[char] {
						usedGroup = true
						break
					}
				}
				if usedGroup && slices.Contains(group, c) {
					conflict = true
					break
				}
			}
			if !conflict {
				valid = true
			}
		}
		bytes[i] = c
		used[c] = true
	}

	return string(bytes)
}
