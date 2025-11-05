package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func NewSessionModel(dtb *db.Database, mdls *Models) *SessionModel {
	return &SessionModel{dtb, mdls}
}

func (self *SessionModel) Create(tx *sql.Tx, ctx context.Context, devId int32, t string, timeSecs int, dataMbytes float64, exp *int, downMbit int, upMbit int, g bool) (*Session, error) {
	var expDays sql.NullInt32
	if exp != nil {
		expDays = sql.NullInt32{Int32: int32(*exp), Valid: true}
	}

	qtx := self.db.Queries.WithTx(tx)
	sId, err := qtx.CreateSession(ctx, queries.CreateSessionParams{
		DeviceID:    devId,
		SessionType: t,
		TimeSecs:    int32(timeSecs),
		DataMbytes:  fmt.Sprintf("%.6f", dataMbytes),
		ExpDays:     expDays,
		DownMbits:   int32(downMbit),
		UpMbits:     int32(upMbit),
		UseGlobal:   g,
	})
	if err != nil {
		log.Println("error creating session:", err)
		return nil, err
	}

	return self.Find(tx, ctx, sId)
}

func (self *SessionModel) Find(tx *sql.Tx, ctx context.Context, id int32) (*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRow, err := qtx.FindSession(ctx, id)
	if err != nil {
		log.Printf("error finding session %v: %v", id, err)
		return nil, err
	}
	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) Update(tx *sql.Tx, ctx context.Context, id int32, devId int32, t string, timeSecs int, dataMbytes float64, timeCons int, dataCons float64, started *time.Time, exp *int, downMbit int, upMbit int, g bool) error {
	var expDays sql.NullInt32
	if exp != nil {
		expDays = sql.NullInt32{Int32: int32(*exp), Valid: true}
	}

	var startedAtTime sql.NullTime
	if started != nil {
		startedAtTime = sql.NullTime{Time: *started, Valid: true}
	}

	types := []string{
		sdkapi.SessionTypeTime,
		sdkapi.SessionTypeData,
		sdkapi.SessionTypeTimeOrData,
	}

	if !sdkutils.SliceContains(types, t) {
		return sdkapi.ErrInvalidSessionType
	}

	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateSession(ctx, queries.UpdateSessionParams{
		DeviceID:        devId,
		SessionType:     t,
		TimeSecs:        int32(timeSecs),
		DataMbytes:      fmt.Sprintf("%.6f", dataMbytes),
		ConsumptionSecs: int32(timeCons),
		ConsumptionMb:   fmt.Sprintf("%.6f", dataCons),
		StartedAt:       startedAtTime,
		ExpDays:         expDays,
		DownMbits:       int32(downMbit),
		UpMbits:         int32(upMbit),
		UseGlobal:       g,
		ID:              id,
	})
	if err != nil {
		log.Printf("error updating session %v: %v", id, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", id)
	return nil
}

func (self *SessionModel) AvailableForDevice(tx *sql.Tx, ctx context.Context, devId int32) (*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRow, err := qtx.FindAvailableSessionForDevice(ctx, devId)
	if err != nil {
		log.Printf("error finding available session for dev %v: %v", devId, err)
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *SessionModel) SessionsForDev(tx *sql.Tx, ctx context.Context, devId int32) ([]*Session, error) {
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
		DownMbits: int32(downMbit),
		UpMbits:   int32(upMbit),
		UseGlobal: g,
	})
	if err != nil {
		log.Println("error updating all bandwidth:", err)
		return err
	}

	log.Println("Successfully updated all bandwidth of valid sessions")
	return nil
}

func (self *SessionModel) Summary(tx *sql.Tx, ctx context.Context, deviceID int32) (*sdkapi.ClientSessionSummary, error) {
	qtx := self.db.Queries.WithTx(tx)

	var remainingSecs int
	var remainingDataMb float64

	summary, err := qtx.SessionSummary(ctx, deviceID)
	if err != nil && errors.Is(sql.ErrNoRows, err) {
		return &sdkapi.ClientSessionSummary{}, nil
	}

	if err != nil {
		return nil, err
	}

	remainingSecs = int(summary.RemainingTimeSecs)
	remainingDataMb = float64(summary.RemainingDataMb)

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs:   remainingSecs,
		RemainingDataMbytes: remainingDataMb,
	}, nil
}
