package models

import (
	"context"
	"database/sql"
	"errors"
	sdkapi "sdk/api"
	"time"

	"core/db"
	"core/db/queries"
	"core/internal/modules/validation"
)

type SessionModel struct {
	db     *db.Database
	models *Models
}

// CreateSessionParams holds parameters for creating a new session
type CreateSessionParams struct {
	UUID           string
	PluginPkg      string
	DeviceID       int64
	Type           sdkapi.SessionType
	TimeSecs       int
	DataMb         float64
	ExpDays        *int
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
}

// UpdateSessionParams holds parameters for updating a session.
// Note: DeviceID is intentionally excluded — it is immutable after creation.
// Only TransferSessionsToDevice (used by device merge) may change it.
type UpdateSessionParams struct {
	ID             int64
	UUID           string
	ProviderPkg    string
	Type           sdkapi.SessionType
	TimeSecs       int
	DataMb         float64
	TimeCons       int
	DataCons       float64
	StartedAt      *time.Time
	ResumedAt      *time.Time
	ExpDays        *int
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
}

func NewSessionModel(dtb *db.Database, mdls *Models) *SessionModel {
	return &SessionModel{dtb, mdls}
}

func (self *SessionModel) Create(ctx context.Context, params CreateSessionParams) (*Session, error) {
	// Validate UUID
	if params.UUID == "" {
		return nil, errors.New("session UUID cannot be empty")
	}

	// Validate session type and bandwidth speeds
	if err := validation.ValidateSessionData(params.Type, params.DownMbits, params.UpMbits, params.UseGlobalSpeed); err != nil {
		return nil, err
	}

	var expDays sql.NullInt64
	if params.ExpDays != nil {
		expDays = sql.NullInt64{Int64: int64(*params.ExpDays), Valid: true}
	}

	sId, err := self.db.Queries.CreateSession(ctx, queries.CreateSessionParams{
		DeviceID:    params.DeviceID,
		Uuid:        params.UUID,
		ProviderPkg: params.PluginPkg,
		SessionType: string(params.Type),
		TimeSecs:    int64(params.TimeSecs),
		DataMbytes:  params.DataMb,
		ExpDays:     expDays,
		DownMbits:   int64(params.DownMbits),
		UpMbits:     int64(params.UpMbits),
		UseGlobal:   params.UseGlobalSpeed,
	})
	if err != nil {
		return nil, err
	}

	return self.Find(ctx, sId)
}

func (self *SessionModel) Find(ctx context.Context, id int64) (*Session, error) {
	sRow, err := self.db.Queries.FindSession(ctx, id)
	if err != nil {
		return nil, err
	}
	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) FindByUUID(ctx context.Context, uuid string) (*Session, error) {
	sRow, err := self.db.Queries.FindSessionByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) Update(ctx context.Context, params UpdateSessionParams) error {
	// Validate session type and bandwidth speeds
	if err := validation.ValidateSessionData(params.Type, params.DownMbits, params.UpMbits, params.UseGlobalSpeed); err != nil {
		return err
	}

	var expDays sql.NullInt64
	if params.ExpDays != nil {
		expDays = sql.NullInt64{Int64: int64(*params.ExpDays), Valid: true}
	}

	var startedAtTime sql.NullTime
	if params.StartedAt != nil {
		startedAtTime = sql.NullTime{Time: *params.StartedAt, Valid: true}
	}

	var resumedAtTime sql.NullTime
	if params.ResumedAt != nil {
		resumedAtTime = sql.NullTime{Time: *params.ResumedAt, Valid: true}
	}

	err := self.db.Queries.UpdateSession(ctx, queries.UpdateSessionParams{
		ProviderPkg:     params.ProviderPkg,
		SessionType:     string(params.Type),
		TimeSecs:        int64(params.TimeSecs),
		DataMbytes:      params.DataMb,
		ConsumptionSecs: int64(params.TimeCons),
		ConsumptionMb:   params.DataCons,
		StartedAt:       startedAtTime,
		ResumedAt:       resumedAtTime,
		ExpDays:         expDays,
		DownMbits:       int64(params.DownMbits),
		UpMbits:         int64(params.UpMbits),
		UseGlobal:       params.UseGlobalSpeed,
		ID:              params.ID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (self *SessionModel) AvailableForDevice(ctx context.Context, devId int64) (*Session, error) {
	sRow, err := self.db.Queries.FindAvailableSessionForDevice(ctx, devId)
	if err != nil {
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) SessionsForDev(ctx context.Context, devId int64) ([]*Session, error) {
	sRows, err := self.db.Queries.FindSessionsForDev(ctx, devId)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, len(sRows))
	for i, s := range sRows {
		sessions[i] = NewSession(self.db, self.models, &s)
	}

	return sessions, nil
}

func (self *SessionModel) UpdateAllBandwidth(ctx context.Context, downMbit int, upMbit int, g bool) error {
	if err := validation.ValidateSessionBandwidth(downMbit, upMbit, g); err != nil {
		return err
	}

	err := self.db.Queries.UpdateAllBandwidth(ctx, queries.UpdateAllBandwidthParams{
		DownMbits: int64(downMbit),
		UpMbits:   int64(upMbit),
		UseGlobal: g,
	})
	if err != nil {
		return err
	}

	return nil
}

func (self *SessionModel) Summary(ctx context.Context, deviceID int64) (*sdkapi.ClientSessionSummary, error) {
	// Get remaining time from time-based sessions
	timeSecs, err := self.db.Queries.SessionSummaryTime(ctx, deviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	remainingSecs := int(timeSecs)
	if remainingSecs < 0 {
		remainingSecs = 0
	}

	// Get remaining data from data-based sessions
	dataMb, err := self.db.Queries.SessionSummaryData(ctx, deviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	remainingDataMb := float64(dataMb)
	if remainingDataMb < 0 {
		remainingDataMb = 0
	}

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs: remainingSecs,
		RemainingDataMb:   remainingDataMb,
	}, nil
}

func (self *SessionModel) Delete(ctx context.Context, id int64) error {
	err := self.db.Queries.DeleteSession(ctx, id)
	if err != nil {
		return err
	}
	return nil
}
