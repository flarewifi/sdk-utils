package connmgr

import (
	"context"
	"log"
	sdkapi "sdk/api"
	"sync"
	"time"

	"core/db"
	"core/db/models"
)

func NewLocalSession(dtb *db.Database, mdls *models.Models, s *models.Session) sdkapi.ISessionSource {
	ls := &LocalSession{db: dtb, mdls: mdls}
	ls.load(s)
	return ls
}

type LocalSession struct {
	mu          sync.RWMutex
	db          *db.Database
	mdls        *models.Models
	id          int64
	devId       int64
	sessionType string
	timeSecs    int
	dataMb      float64
	timeCons    int
	dataCons    float64
	startedAt   *time.Time
	expDays     *int
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
}

func (self *LocalSession) Data() sdkapi.SessionData {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return sdkapi.SessionData{
		Id:             self.id,
		Provider:       "local",
		SessionType:    self.sessionType,
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

func (self *LocalSession) Save(ctx context.Context, data sdkapi.SessionData) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	id := self.id
	devId := self.devId
	t := data.SessionType
	timeSecs := data.TimeSecs
	dataMb := data.DataMb
	timeCons := data.TimeCons
	dataCons := data.DataCons
	started := data.StartedAt
	exp := data.ExpDays
	d := data.DownMbits
	u := data.UpMbits
	g := data.UseGlobalSpeed

	tx, err := self.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = self.mdls.Session().Update(tx, ctx, id, devId, t, timeSecs, dataMb, timeCons, dataCons, started, exp, d, u, g)
	if err != nil {
		log.Println("Session save error: ", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (self *LocalSession) Reload(ctx context.Context) (data sdkapi.SessionData, err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	tx, err := self.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}

	s, err := self.mdls.Session().Find(tx, ctx, self.id)
	if err != nil {
		return self.data(), err
	}

	if err = tx.Commit(); err != nil {
		return
	}

	self.load(s)
	return self.data(), nil
}

func (self *LocalSession) data() sdkapi.SessionData {
	return sdkapi.SessionData{
		Provider:       "local",
		SessionType:    self.sessionType,
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
	self.sessionType = s.SessionType()
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
