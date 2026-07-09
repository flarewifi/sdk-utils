package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"slices"
	"time"

	coreQueries "core/db/queries"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func NewVouchersApi(pluginApi *PluginApi) *VouchersApi {
	v := &VouchersApi{
		pluginApi: pluginApi,
		eventsMgr: pluginApi.EventsMgr,
	}
	pluginApi.VouchersAPI = v
	return v
}

// VouchersApi implements sdkapi.IVouchersApi, scoped per plugin package.
// Event callbacks are stored in the global EventsManager to enable cross-plugin delivery.
type VouchersApi struct {
	pluginApi *PluginApi
	eventsMgr interface {
		EmitVoucherEvent(ctx context.Context, event sdkapi.VoucherEvent, v sdkapi.IVoucher) error
		EmitVoucherBatchEvent(ctx context.Context, event sdkapi.VoucherBatchEvent, batch sdkapi.IVoucherBatch) error
	}
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

// CreateVouchers creates a batch of vouchers and emits EventVoucherBatchCreated.
// Before any DB writes it fires EventVoucherBeforeCreate as a batch event (once)
// and then as a per-voucher event for each voucher inside the transaction.
// Either callback may return an error to cancel creation.
func (self *VouchersApi) CreateVouchers(ctx context.Context, params sdkapi.CreateVouchersParams) ([]sdkapi.IVoucher, error) {
	db := self.pluginApi.db

	if len(params.Entries) == 0 {
		return nil, fmt.Errorf("at least one voucher entry is required")
	}

	// Copy so defaulting below never mutates the caller's slice, and validate
	// each entry's optional Code up front.
	entries := make([]sdkapi.VoucherEntry, len(params.Entries))
	copy(entries, params.Entries)
	seenCodes := make(map[string]bool, len(entries))
	for i := range entries {
		if entries[i].Code != "" {
			if len(entries[i].Code) > 10 {
				return nil, fmt.Errorf("invalid code %q: must be 1-10 characters", entries[i].Code)
			}
			if seenCodes[entries[i].Code] {
				return nil, fmt.Errorf("duplicate code %q in Entries", entries[i].Code)
			}
			seenCodes[entries[i].Code] = true
		}

		// Apply default bandwidth if not specified.
		if entries[i].DownSpeedMbps == 0 {
			entries[i].DownSpeedMbps = 10
		}
		if entries[i].UpSpeedMbps == 0 {
			entries[i].UpSpeedMbps = 10
		}
	}
	params.Entries = entries

	// Guarantee BatchUUID is set before the before-create event so batch-level
	// callbacks (e.g. reseller credit checks) can reference the final UUID.
	if params.BatchUUID == "" {
		params.BatchUUID = generateUUID()
	}

	// Batch-level before-create event: fires once before any DB writes.
	// An error here cancels creation with no rollback needed.
	now := time.Now().UTC()
	previewBatch := &previewVoucherBatch{params: params, providerPkg: self.providerPkg(), now: now}
	if err := self.eventsMgr.EmitVoucherBatchEvent(ctx, sdkapi.EventVoucherBatchBeforeCreate, previewBatch); err != nil {
		return nil, err
	}

	batchUUID := params.BatchUUID

	// Pre-generate each voucher's code/UUID and fire its EventVoucherBeforeCreate
	// here, BEFORE the transaction opens — not inside the RunInTx below like the
	// per-voucher insert loop used to. This app runs SQLite through a single
	// shared connection (db.SetMaxOpenConns(1)): if a subscriber's callback made
	// its own DB call while our transaction held that one connection, it would
	// block forever waiting for a connection only our own (blocked) goroutine
	// could free. Firing here, with no transaction open, means a subscriber's
	// query is safe, and a veto needs no rollback since nothing has been
	// written yet.
	type pendingVoucher struct {
		entry sdkapi.VoucherEntry
		code  string
		uuid  string
	}
	pending := make([]pendingVoucher, len(entries))
	for i, entry := range entries {
		code := entry.Code
		if code == "" {
			code = generateVoucherCode()
		}
		uuid := generateVoucherUUID()
		pending[i] = pendingVoucher{entry: entry, code: code, uuid: uuid}

		previewV := &previewVoucher{
			entry:       entry,
			batchUUID:   batchUUID,
			code:        code,
			uuid:        uuid,
			providerPkg: self.providerPkg(),
			now:         now,
		}
		if err := self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherBeforeCreate, previewV); err != nil {
			return nil, err
		}
	}

	// Wrap batch + voucher creation in a single transaction for atomicity. No
	// events are emitted inside — see the pre-pass above.
	var created []sdkapi.IVoucher
	err := sdkutils.RunInTx(db.DB, ctx, func(tx *sql.Tx) error {
		q := coreQueries.New(tx)

		// Create voucher batch record.
		amount := sql.NullFloat64{}
		if params.Amount != nil {
			amount = sql.NullFloat64{Float64: *params.Amount, Valid: true}
		}
		_, err := q.CreateVoucherBatch(ctx, coreQueries.CreateVoucherBatchParams{
			Uuid:        batchUUID,
			Amount:      amount,
			Metadata:    sql.NullString{},
			ProviderPkg: self.providerPkg(),
		})
		if err != nil {
			return fmt.Errorf("unable to create voucher batch: %w", err)
		}

		for _, p := range pending {
			expiresAt := sql.NullTime{}
			if p.entry.ExpiresAt != nil {
				expiresAt = sql.NullTime{Time: *p.entry.ExpiresAt, Valid: true}
			}
			sessionExpDays := sql.NullInt64{}
			if p.entry.SessionExpDays != nil {
				sessionExpDays = sql.NullInt64{Int64: int64(*p.entry.SessionExpDays), Valid: true}
			}
			useGlobal := int64(0)
			if p.entry.UseGlobal {
				useGlobal = 1
			}
			batchUUIDParam := sql.NullString{String: batchUUID, Valid: true}
			row, err := q.CreateVoucher(ctx, coreQueries.CreateVoucherParams{
				Uuid:           p.uuid,
				Code:           p.code,
				ProviderPkg:    self.providerPkg(),
				SessionType:    string(p.entry.Type),
				TimeSecs:       p.entry.TimeSecs,
				DataMb:         p.entry.DataMb,
				DownSpeedMbps:  p.entry.DownSpeedMbps,
				UpSpeedMbps:    p.entry.UpSpeedMbps,
				SessionExpDays: sessionExpDays,
				UseGlobal:      useGlobal,
				ExpiresAt:      expiresAt,
				BatchUuid:      batchUUIDParam,
			})
			if err != nil {
				return fmt.Errorf("unable to create voucher: %w", err)
			}
			created = append(created, wrapVoucher(row))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Emit generated event after successful commit (outside transaction).
	if len(created) > 0 {
		batch, err := self.FindBatchByUUID(ctx, batchUUID)
		if err == nil {
			self.eventsMgr.EmitVoucherBatchEvent(ctx, sdkapi.EventVoucherBatchCreated, batch)
		}
	}

	return created, nil
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

	self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherUpdated, updated)
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

	// Give subscribers a chance to veto activation before any session is created.
	if err := self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherBeforeActivate, voucher); err != nil {
		return sdkapi.VoucherActivateResult{}, err
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
		UUID:           sessionUUID,
		DevId:          params.Device.ID(),
		Type:           voucher.Type(),
		TimeSecs:       int(voucher.TimeSecs()),
		DataMb:         float64(voucher.DataMb()),
		ExpDays:        voucher.SessionExpDays(),
		DownMbits:      downMbits,
		UpMbits:        upMbits,
		UseGlobalSpeed: useGlobal,
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

	self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherActivated, activated)
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

	self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherDeleted, voucher)
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
		self.eventsMgr.EmitVoucherEvent(ctx, sdkapi.EventVoucherDeleted, v)
	}
	return nil
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

// DeleteBatch removes a voucher batch and all its vouchers by UUID.
// Emits EventVoucherBatchDeleted with the deleted batch.
func (self *VouchersApi) DeleteBatch(ctx context.Context, batchUUID string) error {
	q := coreQueries.New(self.pluginApi.db.DB)

	// Find the batch first to emit event with batch data
	batch, err := self.FindBatchByUUID(ctx, batchUUID)
	if err != nil {
		return fmt.Errorf("unable to find voucher batch: %w", err)
	}

	// Give subscribers a chance to veto the batch deletion before any row is removed.
	if err := self.eventsMgr.EmitVoucherBatchEvent(ctx, sdkapi.EventVoucherBatchBeforeDelete, batch); err != nil {
		return err
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
	self.eventsMgr.EmitVoucherBatchEvent(ctx, sdkapi.EventVoucherBatchDeleted, batch)

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
