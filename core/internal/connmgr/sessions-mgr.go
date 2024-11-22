package connmgr

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"

	"core/internal/db"
	"core/internal/db/models"
	"core/internal/network"
	"core/internal/utils/nftables"
	sdkconnmgr "sdk/api/connmgr"
	sdknet "sdk/api/network"

	sdkslices "github.com/flarehotspot/go-utils/slices"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	EVENT_CONNECTED    string = "session:connected"
	EVENT_DISCONNECTED string = "session:disconnected"
)

func NewSessionsMgr(dtb *db.Database, mdl *models.Models) *SessionsMgr {
	return &SessionsMgr{
		mu:        sync.RWMutex{},
		db:        dtb,
		mdl:       mdl,
		sessions:  []*RunningSession{},
		providers: []sdkconnmgr.ISessionProvider{},
	}
}

type SessionsMgr struct {
	mu        sync.RWMutex
	db        *db.Database
	mdl       *models.Models
	sessions  []*RunningSession
	providers []sdkconnmgr.ISessionProvider
}

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	go func() {
		for data := range trfk.Listen() {
			go func(data *sdknet.TrafficData) {
				self.mu.RLock()
				defer self.mu.RUnlock()

				for _, s := range self.sessions {
					s.UpdateData(data)
				}
			}(&data)
		}
	}()
}

func (self *SessionsMgr) ReloadSessions(ctx context.Context, iface string) error {
	errCh := make(chan error)
	go func() {
		self.mu.RLock()
		defer self.mu.RUnlock()

		for _, rs := range self.sessions {
			lan := rs.Lan()

			if lan.Name() == iface {
				cs := rs.GetSession()
				err := cs.Reload(ctx)
				if err != nil {
					errCh <- err
					break
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					break
				}
			}
		}

		errCh <- nil
	}()
	return <-errCh
}

func (self *SessionsMgr) StopSessions(ctx context.Context, iface string, reason string) {
	done := make(chan bool)
	go func() {
		self.mu.Lock()
		defer self.mu.Unlock()
		defer func() {
			done <- true
		}()

		for _, rs := range self.sessions {
			err := nftables.Disconnect(rs.mac, reason)
			if err != nil {
				log.Println(err)
			}

			lan, err := network.FindByIp(rs.ip)
			if err != nil {
				log.Println(err)
			}

			if lan.Name() == iface {
				rs.Stop(context.Background())
			}
		}
	}()
	<-done
}

func (self *SessionsMgr) Connect(ctx context.Context, clnt sdkconnmgr.IClientDevice, notify string) error {
	errCh := make(chan error)

	go func() {
		if _, ok := self.CurrSession(clnt); ok {
			errCh <- errors.New("Device is already connected.")
			return
		}

		_, err := self.GetSession(ctx, clnt)
		if err != nil {
			errCh <- errors.New("Device has no more available sessions.")
			return
		}

		if !nftables.IsConnected(clnt.MacAddr()) {
			if err := nftables.Connect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
				errCh <- err
				return
			}
		}

		go self.loopSessions(clnt)

		data := map[string]interface{}{"message": notify}

		clnt.Emit(EVENT_CONNECTED, data)
		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) Disconnect(ctx context.Context, clnt sdkconnmgr.IClientDevice, notify string) error {
	err := self.endSession(ctx, clnt)
	if err != nil {
		return err
	}

	data := map[string]interface{}{"message": notify}
	clnt.Emit(EVENT_DISCONNECTED, data)
	return nil
}

func (self *SessionsMgr) IsConnected(clnt sdkconnmgr.IClientDevice) (connected bool) {
	return nftables.IsConnected(clnt.MacAddr())
}

func (self *SessionsMgr) CurrSession(clnt sdkconnmgr.IClientDevice) (cs sdkconnmgr.IClientSession, ok bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	for _, rs := range self.sessions {
		if rs.ClientId() == clnt.Id() {
			return rs.session, true
		}
	}

	return nil, false
}

func (self *SessionsMgr) loopSessions(clnt sdkconnmgr.IClientDevice) {
	ctx := context.Background()

	for nftables.IsConnected(clnt.MacAddr()) {
		errCh := make(chan error)

		go func() {
			cs, err := self.GetSession(ctx, clnt)
			if err != nil {
				errCh <- err
				return
			}

			log.Println("Got new session from : " + cs.Provider())

			self.mu.RLock()
			rs, ok := self.getRunningSession(clnt)
			self.mu.RUnlock()

			if !ok {
				rs, err = NewRunningSession(clnt, cs)
				if err != nil {
					errCh <- err
					return
				}

				err = rs.Start(ctx, cs)
				log.Println("Start session error: ", err)
				if err != nil {
					errCh <- err
					return
				}

				self.mu.Lock()
				self.sessions = append(self.sessions, rs)
				self.mu.Unlock()
			} else {
				err = rs.Start(ctx, cs)
				log.Println("Start session error: ", err)
				if err != nil {
					errCh <- err
					return
				}
			}

			err = <-rs.Done()
			log.Println("Running session is done: ", err)

			errCh <- err
		}()

		err := <-errCh
		log.Println("Session done!!! ", err)

		if err != nil {
			log.Println("Error in session loop: ", err)
			self.Disconnect(ctx, clnt, err.Error())
			return
		}
	}
}

func (self *SessionsMgr) getRunningSession(clnt sdkconnmgr.IClientDevice) (rs *RunningSession, ok bool) {
	for _, rs := range self.sessions {
		if rs.ClientId() == clnt.Id() {
			return rs, true
		}
	}
	return nil, false
}

func (self *SessionsMgr) endSession(ctx context.Context, clnt sdkconnmgr.IClientDevice) error {
	errCh := make(chan error)

	go func() {
		if nftables.IsConnected(clnt.MacAddr()) {
			err := nftables.Disconnect(clnt.IpAddr(), clnt.MacAddr())
			if err != nil {
				errCh <- err
				return
			}
		}

		self.mu.RLock()
		rs, ok := self.getRunningSession(clnt)
		self.mu.RUnlock()

		if ok {
			err := rs.Stop(ctx)
			if err != nil {
				errCh <- err
				return
			}

			err = rs.CleanupTc()
			if err != nil {
				errCh <- err
				return
			}
		}

		self.mu.Lock()
		self.sessions = sdkslices.Filter(self.sessions, func(rs *RunningSession) bool {
			return rs.ClientId() != clnt.Id()
		})
		self.mu.Unlock()

		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) CreateSession(
	ctx context.Context,
	devId pgtype.UUID,
	t uint8,
	timeSecs uint,
	dataMbytes float64,
	expDays *uint,
	downMbits int,
	upMbits int,
	useGlobal bool,
) error {
	_, err := self.mdl.Session().Create(ctx, devId, t, timeSecs, dataMbytes, expDays, downMbits, upMbits, useGlobal)
	return err
}

func (self *SessionsMgr) GetSession(ctx context.Context, clnt sdkconnmgr.IClientDevice) (sdkconnmgr.IClientSession, error) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	if len(self.providers) > 0 {
		for _, provider := range self.providers {
			if remoteSrc, ok := provider.GetSession(ctx, clnt); remoteSrc != nil && ok {
				return NewClientSession(remoteSrc), nil
			}
		}
	}

	localClient := clnt.(*ClientDevice)
	s, err := self.mdl.Session().AvlForDev(ctx, localClient.id)
	if err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return nil, errors.New("No more available sessions")
		}
		return nil, err
	}

	localSrc := NewLocalSession(self.db, self.mdl, s)
	return NewClientSession(localSrc), nil
}

func (self *SessionsMgr) RegisterSessionProvider(provider sdkconnmgr.ISessionProvider) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.providers = append(self.providers, provider)
}
