package connmgr

import (
	"context"
	"log"
	"sync"
	"time"

	"core/internal/db"
	"core/internal/db/models"
	connmgr "sdk/api/connmgr"
	sdkconnmgr "sdk/api/connmgr"

	"github.com/jackc/pgx/v5/pgtype"
)

func NewLocalSession(dtb *db.Database, mdls *models.Models, s *models.Session) connmgr.ISessionSource {
	ls := &LocalSession{db: dtb, mdls: mdls}
	ls.load(s)
	return ls
}

type LocalSession struct {
	mu        sync.RWMutex
	db        *db.Database
	mdls      *models.Models
	id        pgtype.UUID
	devId     pgtype.UUID
	t         uint8
	timeSecs  uint
	dataMb    float64
	timeCons  uint
	dataCons  float64
	startedAt *time.Time
	expDays   *uint
	downMbits int
	upMbits   int
	useGlobal bool
	createdAt time.Time
}

func (self *LocalSession) Data() sdkconnmgr.SessionData {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return sdkconnmgr.SessionData{
		Provider:       "local",
		Type:           self.t,
		TimeSecs:       self.timeSecs,
		DataMb:         self.dataMb,
		TimeCons:       self.timeCons,
		DataCons:       self.dataCons,
		StartedAt:      self.startedAt,
		ExpDays:        self.expDays,
		DownMbits:      self.downMbits,
		UpMbits:        self.upMbits,
		UseGlobalSpeed: self.useGlobal,
		CreatedAt:      self.createdAt,
	}
}

func (self *LocalSession) Save(ctx context.Context, data sdkconnmgr.SessionData) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	id := self.id
	devId := self.devId
	t := data.Type
	timeSecs := data.TimeSecs
	dataMb := data.DataMb
	timeCons := data.TimeCons
	dataCons := data.DataCons
	started := data.StartedAt
	exp := data.ExpDays
	d := data.DownMbits
	u := data.UpMbits
	g := data.UseGlobalSpeed

	err := self.mdls.Session().Update(ctx, id, devId, t, timeSecs, dataMb, timeCons, dataCons, started, exp, d, u, g)
	if err != nil {
		log.Println("Session save error: ", err)
	}

	return err
}

func (self *LocalSession) Reload(ctx context.Context) (sdkconnmgr.SessionData, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	s, err := self.mdls.Session().Find(ctx, self.id)
	if err != nil {
		return self.data(), err
	}

	self.load(s)

	return self.data(), nil
}

func (self *LocalSession) data() sdkconnmgr.SessionData {
	return sdkconnmgr.SessionData{
		Provider:       "local",
		Type:           self.t,
		TimeSecs:       self.timeSecs,
		DataMb:         self.dataMb,
		TimeCons:       self.timeCons,
		DataCons:       self.dataCons,
		StartedAt:      self.startedAt,
		ExpDays:        self.expDays,
		DownMbits:      self.downMbits,
		UpMbits:        self.upMbits,
		UseGlobalSpeed: self.useGlobal,
		CreatedAt:      self.createdAt,
	}
}

func (self *LocalSession) load(s *models.Session) {
	self.id = s.Id()
	self.devId = s.DeviceId()
	self.t = s.SessionType()
	self.timeSecs = s.TimeSecs()
	self.dataMb = s.DataMbyte()
	self.timeCons = s.TimeConsumed()
	self.dataCons = s.DataConsumed()
	self.downMbits = s.DownMbits()
	self.upMbits = s.UpMbits()
	self.useGlobal = s.UseGlobal()
	self.expDays = s.ExpDays()
	self.startedAt = s.StartedAt()
}
