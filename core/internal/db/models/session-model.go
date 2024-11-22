package models

import (
	"context"
	"log"
	"time"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/jackc/pgx/v5/pgtype"
)

type SessionModel struct {
	db     *db.Database
	models *Models
}

func NewSessionModel(dtb *db.Database, mdls *Models) *SessionModel {
	return &SessionModel{dtb, mdls}
}

func (self *SessionModel) Create(ctx context.Context, devId pgtype.UUID, t uint8, timeSecs uint, dataMbytes float64, exp *uint, downMbit int, upMbit int, g bool) (*Session, error) {
	sId, err := self.db.Queries.CreateSession(ctx, sqlc.CreateSessionParams{
		DeviceID:    devId,
		SessionType: int16(t),
		TimeSecs:    pgtype.Int4{Int32: int32(timeSecs)},
		DataMbytes:  pg.Float64ToNumeric(dataMbytes),
		ExpDays:     pgtype.Int4{Int32: int32(*exp)},
		DownMbits:   int32(downMbit),
		UpMbits:     int32(upMbit),
		UseGlobal:   g,
	})
	if err != nil {
		log.Println("error creating session:", err)
		return nil, err
	}

	return self.Find(ctx, sId)
}

func (self *SessionModel) Find(ctx context.Context, id pgtype.UUID) (*Session, error) {
	sRow, err := self.db.Queries.FindSession(ctx, id)
	if err != nil {
		log.Printf("error finding session %v: %v", id, err)
		return nil, err
	}

	expDays := uint(sRow.ExpDays.Int32)

	session := NewSession(self.db, self.models)
	session.id = sRow.ID
	session.deviceId = sRow.DeviceID
	session.timeSecs = uint(sRow.TimeSecs.Int32)
	session.dataMb = pg.NumericToFloat64(sRow.DataMbytes)
	session.timeCons = uint(sRow.ConsumptionSecs.Int32)
	session.dataCons = pg.NumericToFloat64(sRow.ConsumptionMb)
	session.startedAt = &sRow.StartedAt.Time
	session.expDays = &expDays
	// TODO: fix proper expiry calculation
	// session.expiresAt = sRow.ExpiresAt

	session.downMbits = int(sRow.DownMbits)
	session.upMbits = int(sRow.UpMbits)
	session.useGlobal = sRow.UseGlobal
	session.createdAt = sRow.CreatedAt.Time

	return session, nil
}

func (self *SessionModel) Update(ctx context.Context, id pgtype.UUID, devId pgtype.UUID, t uint8, timeSecs uint, dataMbytes float64, timeCons uint, dataCons float64, started *time.Time, exp *uint, downMbit int, upMbit int, g bool) error {
	err := self.db.Queries.UpdateSession(ctx, sqlc.UpdateSessionParams{
		DeviceID:        devId,
		SessionType:     int16(t),
		TimeSecs:        pgtype.Int4{Int32: int32(timeSecs)},
		DataMbytes:      pg.Float64ToNumeric(dataMbytes),
		ConsumptionSecs: pgtype.Int4{Int32: int32(timeCons)},
		ConsumptionMb:   pg.Float64ToNumeric(dataCons),
		StartedAt:       pgtype.Timestamp{Time: *started},
		ExpDays:         pgtype.Int4{Int32: int32(*exp)},
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

func (self *SessionModel) AvlForDev(ctx context.Context, devId pgtype.UUID) (*Session, error) {
	sRow, err := self.db.Queries.FindAvlSessionForDev(ctx, devId)
	if err != nil {
		log.Printf("error finding available session for dev %v: %v", devId, err)
		return nil, err
	}

	expDays := uint(sRow.ExpDays.Int32)

	session := NewSession(self.db, self.models)
	session.id = sRow.ID
	session.deviceId = sRow.DeviceID
	session.sessionType = uint8(sRow.SessionType)
	session.timeSecs = uint(sRow.TimeSecs.Int32)
	session.dataMb = pg.NumericToFloat64(sRow.DataMbytes)
	session.timeCons = uint(sRow.ConsumptionSecs.Int32)
	session.dataCons = pg.NumericToFloat64(sRow.ConsumptionMb)
	session.startedAt = &sRow.StartedAt.Time
	session.expDays = &expDays
	// TODO: proper calculation for expiry date
	// session.expiresAt = sRow.ExpiresAt

	session.downMbits = int(sRow.DownMbits)
	session.upMbits = int(sRow.UpMbits)
	session.useGlobal = sRow.UseGlobal
	session.createdAt = sRow.CreatedAt.Time

	return session, nil
}

func (self *SessionModel) SessionsForDev(ctx context.Context, devId pgtype.UUID) ([]*Session, error) {
	sRows, err := self.db.Queries.FindSessionsForDev(ctx, devId)
	if err != nil {
		log.Println("error finding available sessions for dev:", err)
		return nil, err
	}

	sessions := []*Session{}

	// Parse queried sessions
	for _, s := range sRows {
		expDays := uint(s.ExpDays.Int32)

		sessions = append(sessions, &Session{
			db:          self.db,
			models:      self.models,
			id:          s.ID,
			deviceId:    s.DeviceID,
			sessionType: uint8(s.SessionType),
			timeSecs:    uint(s.TimeSecs.Int32),
			dataMb:      pg.NumericToFloat64(s.DataMbytes),
			timeCons:    uint(s.ConsumptionSecs.Int32),
			dataCons:    pg.NumericToFloat64(s.ConsumptionMb),
			startedAt:   &s.StartedAt.Time,
			expDays:     &expDays,
			// TODO: calculate properly the expiry date
			// expiresAt   *time.Time

			downMbits: int(s.DownMbits),
			upMbits:   int(s.UpMbits),
			useGlobal: s.UseGlobal,
			createdAt: s.CreatedAt.Time,
		})
	}

	return sessions, nil
}

func (self *SessionModel) UpdateAllBandwidth(ctx context.Context, downMbit int, upMbit int, g bool) error {
	err := self.db.Queries.UpdateAllBandwidth(ctx, sqlc.UpdateAllBandwidthParams{
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
