package models

import (
	"context"
	"database/sql"
	"log"
	"time"

	"core/db"
	"core/db/queries"
)

// BuildSessionParams holds parameters for building a Session object from existing data.
type BuildSessionParams struct {
	DB          *db.Database
	Models      *Models
	ID          int64
	UUID        string
	ProviderPkg string
	DeviceID    int64
	SessionType string
	TimeSecs    int
	DataMbytes  float64
	TimeCons    int
	DataCons    float64
	StartedAt   *time.Time
	ResumedAt   *time.Time
	ExpDays     *int
	DownMbits   int
	UpMbits     int
	UseGlobal   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Session struct {
	db          *db.Database
	models      *Models
	id          int64
	uuid        string
	providerPkg string
	deviceId    int64
	sessionType string
	timeSecs    int
	dataMb      float64
	timeCons    int
	dataCons    float64
	startedAt   *time.Time
	resumedAt   *time.Time
	expDays     *int
	expiresAt   *time.Time
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
	updatedAt   time.Time
}

func NewSession(dtb *db.Database, mdls *Models, s *queries.Session) *Session {
	session := &Session{
		db:     dtb,
		models: mdls,
	}

	if s != nil {
		var expDays *int
		if s.ExpDays.Valid {
			val := int(s.ExpDays.Int64)
			expDays = &val
		}

		var startedAt *time.Time
		if s.StartedAt.Valid {
			startedAt = &s.StartedAt.Time
		}

		var resumedAt *time.Time
		if s.ResumedAt.Valid {
			resumedAt = &s.ResumedAt.Time
		}

		session.id = s.ID
		session.uuid = s.Uuid
		session.providerPkg = s.ProviderPkg
		session.deviceId = s.DeviceID
		session.sessionType = s.SessionType
		session.timeSecs = int(s.TimeSecs)
		session.dataMb = s.DataMbytes
		session.timeCons = int(s.ConsumptionSecs)
		session.dataCons = s.ConsumptionMb
		session.expDays = expDays
		session.startedAt = startedAt
		session.resumedAt = resumedAt

		// TODO: fix proper expiry calculation
		// session.expiresAt = sRow.ExpiresAt

		session.downMbits = int(s.DownMbits)
		session.upMbits = int(s.UpMbits)
		session.useGlobal = s.UseGlobal
		session.createdAt = s.CreatedAt
		session.updatedAt = s.UpdatedAt
	}

	return session
}

// BuildSession creates a Session with all fields from session data.
// This is useful for wrapping existing session data without database queries.
// Use this when you have session data from queries and want to create a Session object
// that can be wrapped into IClientSession via NewClientSession().
func BuildSession(params BuildSessionParams) *Session {
	return &Session{
		db:          params.DB,
		models:      params.Models,
		id:          params.ID,
		uuid:        params.UUID,
		providerPkg: params.ProviderPkg,
		deviceId:    params.DeviceID,
		sessionType: params.SessionType,
		timeSecs:    params.TimeSecs,
		dataMb:      params.DataMbytes,
		timeCons:    params.TimeCons,
		dataCons:    params.DataCons,
		startedAt:   params.StartedAt,
		resumedAt:   params.ResumedAt,
		expDays:     params.ExpDays,
		downMbits:   params.DownMbits,
		upMbits:     params.UpMbits,
		useGlobal:   params.UseGlobal,
		createdAt:   params.CreatedAt,
		updatedAt:   params.UpdatedAt,
	}
}

func (self *Session) ID() int64 {
	return self.id
}

func (self *Session) UUID() string {
	return self.uuid
}

func (self *Session) ProviderPkg() string {
	return self.providerPkg
}

func (self *Session) DeviceID() int64 {
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

func (self *Session) ResumedAt() *time.Time {
	return self.resumedAt
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

func (self *Session) UpdatedAt() time.Time {
	return self.updatedAt
}

func (self *Session) Update(ctx context.Context, devId int64, t string, secs int, mb float64, timecon int, datacon float64, started *time.Time, resumed *time.Time, exp *int, downMbit int, upMbit int, g bool) error {
	var startedTime sql.NullTime
	if started != nil {
		startedTime = sql.NullTime{Time: *started, Valid: true}
	}

	var resumedTime sql.NullTime
	if resumed != nil {
		resumedTime = sql.NullTime{Time: *resumed, Valid: true}
	}

	var expDays sql.NullInt64
	if exp != nil {
		expDays = sql.NullInt64{Int64: int64(*exp), Valid: true}
	}

	err := self.db.Queries.UpdateSession(ctx, queries.UpdateSessionParams{
		ProviderPkg:     self.providerPkg,
		DeviceID:        devId,
		SessionType:     t,
		TimeSecs:        int64(secs),
		DataMbytes:      mb,
		ConsumptionSecs: int64(timecon),
		ConsumptionMb:   datacon,
		StartedAt:       startedTime,
		ResumedAt:       resumedTime,
		ExpDays:         expDays,
		DownMbits:       int64(downMbit),
		UpMbits:         int64(upMbit),
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
	self.resumedAt = resumed
	self.downMbits = downMbit
	self.upMbits = upMbit

	return nil
}

func (self *Session) Save(ctx context.Context) error {
	return self.Update(ctx, self.deviceId, self.sessionType, self.timeSecs, self.dataMb, self.timeCons, self.dataCons, self.startedAt, self.resumedAt, self.expDays, self.downMbits, self.upMbits, self.useGlobal)
}
