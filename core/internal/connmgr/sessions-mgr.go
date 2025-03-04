package connmgr

import (
	"context"
	"errors"
	"log"
	"sync"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/utils/nftables"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrSessionQuery = errors.New("Error in session query")
	ErrSessionEmpty = errors.New("Device has no more available sessions.")
)

func NewSessionsMgr(dtb *db.Database, mdl *models.Models) *SessionsMgr {
	return &SessionsMgr{
		mu:        sync.RWMutex{},
		db:        dtb,
		mdl:       mdl,
		sessions:  []*RunningSession{},
		providers: []sdkapi.ISessionProvider{},
	}
}

type SessionsMgr struct {
	mu        sync.RWMutex
	db        *db.Database
	mdl       *models.Models
	sessions  []*RunningSession
	providers []sdkapi.ISessionProvider
}

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	go func() {
		for data := range trfk.Listen() {
			go func(data *sdkapi.TrafficData) {
				self.mu.RLock()
				defer self.mu.RUnlock()

				for _, s := range self.sessions {
					s.UpdateDataConsumption(data)
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

func (self *SessionsMgr) Connect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	errCh := make(chan error)

	go func() {
		if _, ok := self.CurrSession(clnt); ok {
			errCh <- errors.New("Device is already connected.")
			return
		}

		tx, err := self.db.SqlDB().Begin(ctx)
		if err != nil {
			errCh <- ErrSessionQuery
			return
		}

		_, err = self.GetSession(tx, ctx, clnt)
		if err != nil {
			errCh <- ErrSessionEmpty
			return
		}

		if err := tx.Commit(ctx); err != nil {
			errCh <- ErrSessionQuery
			return
		}

		if !nftables.IsConnected(clnt.MacAddr()) {
			if err := nftables.Connect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
				errCh <- err
				return
			}
		}

		go self.loopSessions(clnt)

		clnt.Emit(sdkapi.EventSessionConnected, []byte(notify))
		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) Disconnect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	err := self.endSession(ctx, clnt)
	if err != nil {
		return err
	}

	clnt.Emit(sdkapi.EventSessionDisconnected, []byte(notify))
	return nil
}

func (self *SessionsMgr) IsConnected(clnt sdkapi.IClientDevice) (connected bool) {
	return nftables.IsConnected(clnt.MacAddr())
}

func (self *SessionsMgr) CurrSession(clnt sdkapi.IClientDevice) (cs sdkapi.IClientSession, ok bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	for _, rs := range self.sessions {
		if rs.ClientId() == clnt.Id() {
			return rs.session, true
		}
	}

	return nil, false
}

func (self *SessionsMgr) loopSessions(clnt sdkapi.IClientDevice) {
	ctx := context.Background()

	for nftables.IsConnected(clnt.MacAddr()) {
		errCh := make(chan error)

		go func() {
			tx, err := self.db.SqlDB().Begin(ctx)
			if err != nil {
				errCh <- err
				return
			}

			cs, err := self.GetSession(tx, ctx, clnt)
			if err != nil {
				errCh <- err
				return
			}

			if err = tx.Commit(ctx); err != nil {
				errCh <- err
				return
			}

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

func (self *SessionsMgr) getRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	for _, rs := range self.sessions {
		if rs.ClientId() == clnt.Id() {
			return rs, true
		}
	}
	return nil, false
}

func (self *SessionsMgr) endSession(ctx context.Context, clnt sdkapi.IClientDevice) error {
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
		self.sessions = sdkutils.SliceFilter(self.sessions, func(rs *RunningSession) bool {
			return rs.ClientId() != clnt.Id()
		})
		self.mu.Unlock()

		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) CreateSession(
	tx pgx.Tx,
	ctx context.Context,
	devId pgtype.UUID,
	t string,
	timeSecs int,
	dataMbytes float64,
	expDays *int,
	downMbits int,
	upMbits int,
	useGlobal bool,
) error {
	_, err := self.mdl.Session().Create(tx, ctx, devId, t, timeSecs, dataMbytes, expDays, downMbits, upMbits, useGlobal)
	return err
}

func (self *SessionsMgr) GetSession(tx pgx.Tx, ctx context.Context, clnt sdkapi.IClientDevice) (sdkapi.IClientSession, error) {
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
	s, err := self.mdl.Session().AvailableForDevice(tx, ctx, localClient.id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("No more available sessions")
		}
		return nil, err
	}

	localSrc := NewLocalSession(self.db, self.mdl, s)
	return NewClientSession(localSrc), nil
}

// SessionSummary
func (self *SessionsMgr) SessionSummary(tx pgx.Tx, ctx context.Context, clnt sdkapi.IClientDevice) (*sdkapi.ClientSessionSummary, error) {
	summary, err := self.mdl.Session().Summary(tx, ctx, clnt.Id())
	if err != nil {
		return nil, err
	}

	rs, ok := self.getRunningSession(clnt)
	if !ok {
		return summary, nil
	}

	timeDiff, mbDiff := rs.Diff()
	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs:   summary.RemainingTimeSecs - timeDiff,
		RemainingDataMbytes: summary.RemainingDataMbytes - mbDiff,
	}, nil
}

func (self *SessionsMgr) RegisterSessionProvider(provider sdkapi.ISessionProvider) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.providers = append(self.providers, provider)
}
