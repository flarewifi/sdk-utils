package models

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	db          *db.Database
	models      *Models
	id          pgtype.UUID
	deviceId    pgtype.UUID
	sessionType string
	timeSecs    int
	dataMb      float64
	timeCons    int
	dataCons    float64
	startedAt   *time.Time
	expDays     *int
	expiresAt   *time.Time
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
}

func NewSession(dtb *db.Database, mdls *Models, s *queries.Session) *Session {

	session := &Session{
		db:     dtb,
		models: mdls,
	}

	if s != nil {
		var expDays *int
		if s.ExpDays.Valid {
			val := int(s.ExpDays.Int32)
			expDays = &val
		}

		var startedAt *time.Time
		if s.StartedAt.Valid {
			startedAt = &s.StartedAt.Time
		}

		session.id = s.ID
		session.deviceId = s.DeviceID
		session.timeSecs = int(s.TimeSecs)
		session.dataMb = sdkutils.PgNumericToFloat64(s.DataMbytes)
		session.timeCons = int(s.ConsumptionSecs)
		session.dataCons = sdkutils.PgNumericToFloat64(s.ConsumptionMb)
		session.expDays = expDays
		session.startedAt = startedAt

		// TODO: fix proper expiry calculation
		// session.expiresAt = sRow.ExpiresAt

		session.downMbits = int(s.DownMbits)
		session.upMbits = int(s.UpMbits)
		session.useGlobal = s.UseGlobal

		if s.CreatedAt.Valid {
			session.createdAt = s.CreatedAt.Time
		}
	}

	return session
}

func BuildSession(id pgtype.UUID, devId pgtype.UUID, t string, timeSecs int, dataMb float64, timeCons int, dataCons float64, startedAt *time.Time, expDays *int, expiresAt *time.Time, dmbits int, umbits int, g bool) *Session {
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

func (self *Session) SessionType() string {
	return self.sessionType
}

func (self *Session) TimeSecs() int {
	return self.timeSecs
}

func (self *Session) DataMbyte() float64 {
	return self.dataMb
}

func (self *Session) TimeConsumed() int {
	return self.timeCons
}

func (self *Session) DataConsumed() float64 {
	return self.dataCons
}

func (self *Session) StartedAt() *time.Time {
	return self.startedAt
}

func (self *Session) ExpDays() *int {
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

func (self *Session) Update(ctx context.Context, devId pgtype.UUID, t string, secs int, mb float64, timecon int, datacon float64, started *time.Time, exp *int, downMbit int, upMbit int, g bool) error {
	var startedTime pgtype.Timestamp
	if started != nil {
		startedTime = pgtype.Timestamp{Time: *started, Valid: true}
	}

	var expDays pgtype.Int4
	if exp != nil {
		expDays = pgtype.Int4{Int32: int32(*exp), Valid: true}
	}

	err := self.db.Queries.UpdateSession(ctx, queries.UpdateSessionParams{
		DeviceID:        devId,
		SessionType:     t,
		TimeSecs:        int32(secs),
		DataMbytes:      sdkutils.PgFloat64ToNumeric(mb),
		ConsumptionSecs: int32(timecon),
		ConsumptionMb:   sdkutils.PgFloat64ToNumeric(datacon),
		StartedAt:       startedTime,
		ExpDays:         expDays,
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
