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

const (
	SessionTypeTime uint8 = iota
	SessionTypeData
	SessionTypeTimeOrData
)

type Session struct {
	db          *db.Database
	models      *Models
	id          pgtype.UUID
	deviceId    pgtype.UUID
	sessionType uint8
	timeSecs    uint
	dataMb      float64
	timeCons    uint
	dataCons    float64
	startedAt   *time.Time
	expDays     *uint
	expiresAt   *time.Time
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
}

func NewSession(dtb *db.Database, mdls *Models) *Session {
	return &Session{
		db:     dtb,
		models: mdls,
	}
}

func BuildSession(id pgtype.UUID, devId pgtype.UUID, t uint8, timeSecs uint, dataMb float64, timeCons uint, dataCons float64, startedAt *time.Time, expDays *uint, expiresAt *time.Time, dmbits int, umbits int, g bool) *Session {
	return &Session{
		id:          id,
		deviceId:    devId,
		sessionType: t,
		timeSecs:    timeSecs,
		dataMb:      dataMb,
		timeCons:    timeCons,
		dataCons:    dataCons,
		startedAt:   startedAt,
		expDays:     expDays,
		expiresAt:   expiresAt,
		downMbits:   dmbits,
		upMbits:     umbits,
		useGlobal:   g,
	}
}

func (self *Session) Id() pgtype.UUID {
	return self.id
}

func (self *Session) DeviceId() pgtype.UUID {
	return self.deviceId
}

func (self *Session) SessionType() uint8 {
	return self.sessionType
}

func (self *Session) TimeSecs() uint {
	return self.timeSecs
}

func (self *Session) DataMbyte() float64 {
	return self.dataMb
}

func (self *Session) TimeConsumed() uint {
	return self.timeCons
}

func (self *Session) DataConsumed() float64 {
	return self.dataCons
}

func (self *Session) StartedAt() *time.Time {
	return self.startedAt
}

func (self *Session) ExpDays() *uint {
	return self.expDays
}

func (self *Session) CretedAt() time.Time {
	return self.createdAt
}

func (self *Session) ExpiresAt() *time.Time {
	if self.startedAt != nil && self.expDays != nil {
		exp := self.startedAt.Add(time.Hour * 24 * time.Duration(*self.expDays))
		return &exp
	}
	return nil
}

func (self *Session) DownMbits() int {
	return self.downMbits
}

func (self *Session) UpMbits() int {
	return self.upMbits
}

func (self *Session) UseGlobal() bool {
	return self.useGlobal
}

func (self *Session) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Session) Update(ctx context.Context, devId pgtype.UUID, t uint8, secs uint, mb float64, timecon uint, datacon float64, started *time.Time, exp *uint, downMbit int, upMbit int, g bool) error {
	err := self.db.Queries.UpdateSession(ctx, sqlc.UpdateSessionParams{
		DeviceID:        devId,
		SessionType:     int16(t),
		TimeSecs:        pgtype.Int4{Int32: int32(secs)},
		DataMbytes:      pg.Float64ToNumeric(mb),
		ConsumptionSecs: pgtype.Int4{Int32: int32(timecon)},
		ConsumptionMb:   pg.Float64ToNumeric(datacon),
		StartedAt:       pgtype.Timestamp{Time: *started},
		ExpDays:         pgtype.Int4{Int32: int32(*exp)},
		DownMbits:       int32(downMbit),
		UpMbits:         int32(upMbit),
		UseGlobal:       g,
		ID:              self.id,
	})
	if err != nil {
		log.Printf("error updating session %v: %v", self.id, err)
		return err
	}

	self.deviceId = devId
	self.sessionType = t
	self.timeSecs = secs
	self.dataMb = mb
	self.timeCons = timecon
	self.dataCons = datacon
	self.startedAt = started
	self.downMbits = downMbit
	self.upMbits = upMbit

	return nil
}

func (self *Session) Save(ctx context.Context) error {
	return self.Update(ctx, self.deviceId, self.sessionType, self.timeSecs, self.dataMb, self.timeCons, self.dataCons, self.startedAt, self.expDays, self.downMbits, self.upMbits, self.useGlobal)
}
