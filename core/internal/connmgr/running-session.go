package connmgr

import (
	"context"
	"log"
	"sync"
	"time"

	"core/internal/db/models"
	"core/internal/network"
	jobque "core/internal/utils/job-que"
	"core/internal/utils/tc"
	connmgr "sdk/api/connmgr"
	sdkconnmgr "sdk/api/connmgr"
	sdknet "sdk/api/network"

	"github.com/jackc/pgx/v5/pgtype"
)

var sessionQ *jobque.JobQue = jobque.NewJobQue()

func NewRunningSession(clnt sdkconnmgr.IClientDevice, s connmgr.IClientSession) (*RunningSession, error) {
	lan, err := network.FindByIp(clnt.IpAddr())
	if err != nil {
		return nil, err
	}

	rs := RunningSession{
		session:   s,
		clntId:    clnt.Id(),
		ip:        clnt.IpAddr(),
		mac:       clnt.MacAddr(),
		lan:       lan,
		callbacks: []chan error{},
	}

	return &rs, nil
}

type RunningSession struct {
	mu         sync.RWMutex
	clntId     pgtype.UUID
	ip         string
	mac        string
	lan        *network.NetworkLan
	tcClassId  *tc.TcClassId
	tcFilter   *tc.TcFilter
	timeTicker *time.Ticker
	tickerDone chan bool
	session    connmgr.IClientSession
	callbacks  []chan error
}

func (self *RunningSession) ClientId() pgtype.UUID {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.clntId
}

func (self *RunningSession) GetSession() connmgr.IClientSession {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.session
}

func (self *RunningSession) Lan() *network.NetworkLan {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.lan
}

func (self *RunningSession) Done() <-chan error {
	self.mu.Lock()
	defer self.mu.Unlock()

	ch := make(chan error)
	self.callbacks = append(self.callbacks, ch)
	return ch
}

func (self *RunningSession) Start(ctx context.Context, s connmgr.IClientSession) error {
	_, err := sessionQ.Exec(func() (interface{}, error) {
		self.mu.Lock()
		defer self.mu.Unlock()

		self.session = s

		if s.StartedAt() == nil {
			started := time.Now()
			s.SetStartedAt(&started)

			if err := s.Save(ctx); err != nil {
				return nil, err
			}
		}

		if self.tcClassId == nil {
			if err := self.initTc(); err != nil {
				return nil, err
			}
		} else {
			if err := self.updateTc(); err != nil {
				return nil, err
			}
		}

		if self.timeTicker == nil {
			self.initTimeTicker()
			log.Println("Session Tick has started...")
		}

		return nil, nil
	})

	return err
}

func (self *RunningSession) Stop(ctx context.Context) error {
	_, err := sessionQ.Exec(func() (interface{}, error) {
		self.mu.Lock()
		defer self.mu.Unlock()

		err := self.save(ctx)
		self.cleanUpTick()
		self.runCallbacks(err)

		return nil, nil
	})

	return err
}

func (self *RunningSession) CleanupTc() error {
	errCh := make(chan error)

	go func() {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.tcClassId != nil {
			log.Println("Clean up TC...")
			classid := self.tcClassId.Uint()

			err := self.lan.DelFilter(self.ip, classid)
			if err != nil {
				errCh <- err
				return
			}

			err = self.lan.DelClass(classid)
			self.tcClassId = nil

			errCh <- err
			return
		}

		log.Println("Done cleaning TC.")
		errCh <- nil
	}()

	return <-errCh
}

func (self *RunningSession) UpdateData(stats *sdknet.TrafficData) {
	self.mu.Lock()
	defer self.mu.Unlock()

	download, dlok := stats.Download[self.ip]
	upload, ulok := stats.Upload[self.mac]

	if dlok && ulok {
		dataconMb := float64(download.Bytes+upload.Bytes) / (1 * 1000 * 1000)
		log.Println("CONSUMPTION MB: ", dataconMb)
		self.session.IncDataCons(dataconMb)

		if self.isConsumed() {
			log.Println("Session data is consumed!!!")
			go self.Stop(context.Background())
		}
	}
}

func (self *RunningSession) initTimeTicker() {
	tickerCh := make(chan bool)
	ticker := time.NewTicker(time.Second)

	self.timeTicker = ticker
	self.tickerDone = tickerCh

	go func() {
		self.mu.RLock()
		s := self.session
		self.mu.RUnlock()

		for {
			select {
			case <-tickerCh:
				return
			case <-ticker.C:
				go func() {
					self.mu.RLock()
					defer self.mu.RUnlock()

					s.IncTimeCons(1)

					log.Println("time tick...")

					// save every 15s
					if s.TimeConsumption()%15 == 0 {
						err := self.save(context.Background())
						if err != nil {
							log.Println(err)
							go self.Stop(context.Background())
							return
						}
					}

					if self.isConsumed() {
						log.Println("Session time is consumed!!!")
						go self.Stop(context.Background())
					}
				}()
			}
		}
	}()
}

func (self *RunningSession) initTc() error {
	classid := tc.GetAvailableId()
	defer classid.Cancel()

	lan := self.lan
	s := self.session

	err := lan.CreateClass(classid.Uint(), s.DownMbits(), s.UpMbits())
	if err != nil {
		return err
	}

	err = lan.CreateFilter(self.ip, classid.Uint())
	if err != nil {
		lan.DelClass(classid.Uint())
		return err
	}

	classid.Commit()
	self.tcClassId = &classid

	return nil
}

func (self *RunningSession) updateTc() error {
	var (
		downMbits = self.session.DownMbits()
		upMbits   = self.session.UpMbits()
		useGlobal = self.session.UseGlobalSpeed()
	)

	if useGlobal {
		lan, err := network.FindByIp(self.ip)
		if err != nil {
			return err
		}

		d, u := lan.Bandwidth()
		downMbits, upMbits = int(d), int(u)
	}

	return self.lan.ChangeClass(self.tcClassId.Uint(), downMbits, upMbits)
}

func (self *RunningSession) cleanUpTick() {
	log.Println("Cleaning up session tick...")
	if self.timeTicker != nil {
		self.timeTicker.Stop()
		self.timeTicker = nil
		self.tickerDone <- true
		self.tickerDone = nil
	}
	log.Println("Done cleaning session tick.")
}

func (self *RunningSession) save(ctx context.Context) error {
	if self.session != nil {
		if err := self.session.Save(ctx); err != nil {
			return err
		}

		if err := self.session.Reload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (self *RunningSession) runCallbacks(err error) {
	for _, cb := range self.callbacks {
		cb <- err
	}
	self.callbacks = []chan error{}
	log.Println("Done running callbacks.")
}

func (self *RunningSession) expired() (ok bool) {
	expiresAt := self.session.ExpiresAt()
	if expiresAt != nil {
		return !time.Now().Before(*expiresAt)
	}
	return false
}

func (self *RunningSession) isConsumed() bool {
	s := self.session
	t := s.Type()

	if t == models.SessionTypeTime || t == models.SessionTypeTimeOrData {
		isTimeConsumed := s.TimeConsumption() >= s.TimeSecs()
		return isTimeConsumed || self.expired()
	}

	if t == models.SessionTypeData || t == models.SessionTypeTimeOrData {
		isDataConsumed := s.DataConsumption() >= s.DataMb()
		return isDataConsumed || self.expired()
	}

	return false
}
