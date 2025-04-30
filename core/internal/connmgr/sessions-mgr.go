package connmgr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/utils/nftables"
	sdkapi "sdk/api"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrSessionQuery = errors.New("Error in session query")
	ErrSessionEmpty = errors.New("Device has no more available sessions.")
)

func NewSessionsMgr(dtb *db.Database, mdl *models.Models) *SessionsMgr {
	sessionMgr := &SessionsMgr{
		db:        dtb,
		mdl:       mdl,
		sessions:  sync.Map{},
		providers: []sdkapi.ISessionProvider{},
	}
	return sessionMgr
}

type SessionsMgr struct {
	db        *db.Database
	mdl       *models.Models
	sessions  sync.Map
	providers []sdkapi.ISessionProvider
}

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	go func() {
		for data := range trfk.Listen() {
			go func(data *sdkapi.TrafficData) {
				self.sessions.Range(func(key, value any) bool {
					rs := value.(*RunningSession)
					rs.UpdateDataConsumption(data)
					return true
				})
			}(&data)
		}
	}()
}

func (self *SessionsMgr) ReloadSessions(ctx context.Context, iface string) error {
	errCh := make(chan error)

	go func() {
		self.sessions.Range(func(key, value any) bool {
			rs := value.(*RunningSession)
			lan := rs.Lan()

			if lan.Name() == iface {
				cs := rs.GetSession()
				err := cs.Reload(ctx)
				if err != nil {
					errCh <- err
					return false
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					return false
				}
			}

			return true
		})

		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) StopSessions(ctx context.Context, iface string, reason string) {
	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		err := nftables.Disconnect(rs.mac, reason)
		if err != nil {
			log.Println(err)
		}

		lan, err := network.FindByIp(rs.ip)
		if err != nil {
			log.Println(err)
		}

		if lan.Name() == iface {
			rs.Stop(ctx)
		}

		return true
	})
}

func (self *SessionsMgr) Connect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	errReturnCh := make(chan error)

	if clnt.Status() == sdkapi.Blocked {
		return errors.New("Device is blocked.")
	}

	go func() {
		if _, ok := self.CurrSession(clnt); ok {
			errReturnCh <- errors.New("Device is already connected.")
			return
		}

		_, err := self.GetSession(ctx, clnt)
		if err != nil {
			errReturnCh <- ErrSessionEmpty
			return
		}

		if !nftables.IsConnected(clnt.MacAddr()) {
			if err := nftables.Connect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
				errReturnCh <- err
				return
			}
		}

		startCh := make(chan error)
		go self.loopSessions(startCh, clnt)

		err = <-startCh
		close(startCh)

		if err != nil {
			errReturnCh <- err
			return
		}

		clnt.Emit(sdkapi.EventSessionConnected, []byte(notify))
		errReturnCh <- nil
	}()

	err := <-errReturnCh
	if err == nil {
		tx, err := self.db.SqlDB().Begin(ctx)
		if err != nil {
			return fmt.Errorf("unble to create db pool: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := clnt.Update(
			tx, ctx, clnt.MacAddr(), clnt.IpAddr(), clnt.Hostname(), int(sdkapi.Connected),
		); err != nil {
			return fmt.Errorf("unable to update device status to connected: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("unable to commit db transaction: %w", err)
		}
	}

	return err
}

func (self *SessionsMgr) Disconnect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	err := self.endSession(ctx, clnt)
	if err != nil {
		return err
	}

	clnt.Emit(sdkapi.EventSessionDisconnected, []byte(notify))

	tx, err := self.db.SqlDB().Begin(ctx)
	if err != nil {
		return fmt.Errorf("unble to create db pool: %w", err)
	}
	defer tx.Rollback(ctx)

	err = clnt.Update(tx, ctx, clnt.MacAddr(), clnt.IpAddr(), clnt.Hostname(), int(sdkapi.Disconnected))
	if err != nil {
		return fmt.Errorf("unable to update device status to disconnected: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit db transaction: %w", err)
	}

	return nil
}

func (self *SessionsMgr) IsConnected(clnt sdkapi.IClientDevice) (connected bool) {
	return nftables.IsConnected(clnt.MacAddr())
}

func (self *SessionsMgr) CurrSession(clnt sdkapi.IClientDevice) (cs sdkapi.IClientSession, ok bool) {
	v, ok := self.sessions.Load(clnt.Id())
	if !ok {
		return nil, false
	}

	rs, ok := v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs.session, true
}

func (self *SessionsMgr) loopSessions(resultCh chan<- error, clnt sdkapi.IClientDevice) {
	var callbackDone atomic.Bool
	ctx := context.Background()

	for nftables.IsConnected(clnt.MacAddr()) {
		errCh := make(chan error)

		go func() {
			cs, err := self.GetSession(ctx, clnt)
			if err != nil {
				errCh <- err
				return
			}

			rs, ok := self.getRunningSession(clnt)
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

				self.sessions.Store(clnt.Id(), rs)
			} else {
				err = rs.Start(ctx, cs)
				log.Println("Start session error: ", err)
				if err != nil {
					errCh <- err
					return
				}
			}

			// Start was successful
			if !callbackDone.Load() {
				resultCh <- nil
				callbackDone.Store(true)
			}

			err = <-rs.Done()
			errCh <- err
		}()

		err := <-errCh

		if !callbackDone.Load() {
			resultCh <- err
			callbackDone.Store(true)
		}

		if err != nil {
			log.Println("Error in session loop: ", err)
			self.Disconnect(ctx, clnt, err.Error())
			return
		}
	}
}

func (self *SessionsMgr) getRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	v, ok := self.sessions.Load(clnt.Id())
	if !ok {
		return nil, false
	}

	rs, ok = v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs, true
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

		rs, ok := self.getRunningSession(clnt)

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

		self.sessions.Delete(clnt.Id())

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

func (self *SessionsMgr) GetSession(ctx context.Context, clnt sdkapi.IClientDevice) (sdkapi.IClientSession, error) {
	if len(self.providers) > 0 {
		for _, provider := range self.providers {
			if remoteSrc, ok := provider.GetSession(ctx, clnt); remoteSrc != nil && ok {
				return NewClientSession(remoteSrc), nil
			}
		}
	}

	tx, err := self.db.SqlDB().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	localClient := clnt.(*ClientDevice)
	s, err := self.mdl.Session().AvailableForDevice(tx, ctx, localClient.id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("No more available sessions")
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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
	self.providers = append(self.providers, provider)
}
