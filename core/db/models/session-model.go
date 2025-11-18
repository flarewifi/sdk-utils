package models

import (
	"context"
	"database/sql"
	"errors"
	"log"
	sdkapi "sdk/api"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type SessionModel struct {
	db     *db.Database
	models *Models
}

// CreateSessionParams holds parameters for creating a new session
type CreateSessionParams struct {
	UID         string
	PluginPkg   string
	DeviceID    int64
	SessionType sdkapi.SessionType
	TimeSecs    int
	DataMbytes  float64
	ExpDays     *int
	DownMbits   int
	UpMbits     int
	UseGlobal   bool
}

// UpdateSessionParams holds parameters for updating a session
type UpdateSessionParams struct {
	ID          int64
	UID         string
	ProviderPkg string
	DeviceID    int64
	SessionType sdkapi.SessionType
	TimeSecs    int
	DataMbytes  float64
	TimeCons    int
	DataCons    float64
	StartedAt   *time.Time
	ExpDays     *int
	DownMbits   int
	UpMbits     int
	UseGlobal   bool
}

func NewSessionModel(dtb *db.Database, mdls *Models) *SessionModel {
	return &SessionModel{dtb, mdls}
}

func (self *SessionModel) Create(tx *sql.Tx, ctx context.Context, params CreateSessionParams) (*Session, error) {
	var expDays sql.NullInt64
	if params.ExpDays != nil {
		expDays = sql.NullInt64{Int64: int64(*params.ExpDays), Valid: true}
	}

	qtx := self.db.Queries.WithTx(tx)
	sId, err := qtx.CreateSession(ctx, queries.CreateSessionParams{
		DeviceID:    params.DeviceID,
		Uid:         params.UID,
		ProviderPkg: params.PluginPkg,
		SessionType: string(params.SessionType),
		TimeSecs:    int64(params.TimeSecs),
		DataMbytes:  sql.NullFloat64{Float64: params.DataMbytes, Valid: true},
		ExpDays:     expDays,
		DownMbits:   int64(params.DownMbits),
		UpMbits:     int64(params.UpMbits),
		UseGlobal:   params.UseGlobal,
	})
	if err != nil {
		log.Println("error creating session:", err)
		return nil, err
	}

	return self.Find(tx, ctx, sId)
}

func (self *SessionModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRow, err := qtx.FindSession(ctx, id)
	if err != nil {
		log.Printf("error finding session %v: %v", id, err)
		return nil, err
	}
	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) Update(tx *sql.Tx, ctx context.Context, params UpdateSessionParams) error {
	var expDays sql.NullInt64
	if params.ExpDays != nil {
		expDays = sql.NullInt64{Int64: int64(*params.ExpDays), Valid: true}
	}

	var startedAtTime sql.NullTime
	if params.StartedAt != nil {
		startedAtTime = sql.NullTime{Time: *params.StartedAt, Valid: true}
	}

	types := []sdkapi.SessionType{
		sdkapi.SessionTypeTime,
		sdkapi.SessionTypeData,
		sdkapi.SessionTypeTimeOrData,
	}

	if !sdkutils.SliceContains(types, params.SessionType) {
		return errors.New("invalid session type")
	}

	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateSession(ctx, queries.UpdateSessionParams{
		DeviceID:        params.DeviceID,
		SessionType:     string(params.SessionType),
		TimeSecs:        int64(params.TimeSecs),
		DataMbytes:      sql.NullFloat64{Float64: params.DataMbytes, Valid: true},
		ConsumptionSecs: int64(params.TimeCons),
		ConsumptionMb:   params.DataCons,
		StartedAt:       startedAtTime,
		ExpDays:         expDays,
		DownMbits:       int64(params.DownMbits),
		UpMbits:         int64(params.UpMbits),
		UseGlobal:       params.UseGlobal,
		ID:              params.ID,
	})
	if err != nil {
		log.Printf("error updating session %v: %v", params.ID, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", params.ID)
	return nil
}

func (self *SessionModel) AvailableForDevice(ctx context.Context, devId int64) (*Session, error) {
	sRow, err := self.db.Queries.FindAvailableSessionForDevice(ctx, devId)
	if err != nil {
		log.Printf("error finding available session for dev %v: %v", devId, err)
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) SessionsForDev(tx *sql.Tx, ctx context.Context, devId int64) ([]*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRows, err := qtx.FindSessionsForDev(ctx, devId)
	if err != nil {
		log.Println("error finding available sessions for dev:", err)
		return nil, err
	}

	sessions := make([]*Session, len(sRows))
	for i, s := range sRows {
		sessions[i] = NewSession(self.db, self.models, &s)
	}

	return sessions, nil
}

func (self *SessionModel) UpdateAllBandwidth(tx *sql.Tx, ctx context.Context, downMbit int, upMbit int, g bool) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateAllBandwidth(ctx, queries.UpdateAllBandwidthParams{
		DownMbits: int64(downMbit),
		UpMbits:   int64(upMbit),
		UseGlobal: g,
	})
	if err != nil {
		log.Println("error updating all bandwidth:", err)
		return err
	}

	log.Println("Successfully updated all bandwidth of valid sessions")
	return nil
}

func (self *SessionModel) Summary(ctx context.Context, deviceID int64) (*sdkapi.ClientSessionSummary, error) {
	var remainingSecs int
	var remainingDataMb float64

	summary, err := self.db.Queries.SessionSummary(ctx, deviceID)
	if err != nil && errors.Is(sql.ErrNoRows, err) {
		return &sdkapi.ClientSessionSummary{}, nil
	}

	if err != nil {
		return nil, err
	}

	remainingSecs = int(summary.RemainingTimeSecs)
	if remainingSecs < 0 {
		remainingSecs = 0
	}
	remainingDataMb = float64(summary.RemainingDataMb)
	if remainingDataMb < 0 {
		remainingDataMb = 0
	}

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs:   remainingSecs,
		RemainingDataMbytes: remainingDataMb,
	}, nil
}
