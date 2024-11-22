package connmgr

import (
	"fmt"
	"log"
	"net/http"

	"core/internal/db"
	"core/internal/db/models"
	jobque "core/internal/utils/job-que"
	connmgr "sdk/api/connmgr"

	"github.com/jackc/pgx/v5"
)

const (
	EVENT_CLIENT_CREATED = "client:created"
	EVENT_CLIENT_CHANGED = "client:changed"
)

var (
	regQue *jobque.JobQue = jobque.NewJobQue()
)

func NewClientRegister(dtb *db.Database, mdls *models.Models) *ClientRegister {
	return &ClientRegister{
		db:           dtb,
		mdls:         mdls,
		createdHooks: []connmgr.ClientCreatedHookFn{},
		changedHooks: []connmgr.ClientChangedHookFn{},
	}
}

type ClientRegister struct {
	db           *db.Database
	mdls         *models.Models
	mgr          *SessionsMgr
	createdHooks []connmgr.ClientCreatedHookFn
	changedHooks []connmgr.ClientChangedHookFn
}

func (reg *ClientRegister) ClientCreatedHook(fn ...connmgr.ClientCreatedHookFn) {
	reg.createdHooks = append(reg.createdHooks, fn...)
}

func (reg *ClientRegister) ClientChangedHook(fn ...connmgr.ClientChangedHookFn) {
	reg.changedHooks = append(reg.changedHooks, fn...)
}

func (reg *ClientRegister) Register(r *http.Request, mac string, ip string, hostname string) (connmgr.IClientDevice, error) {
	ctx := r.Context()
	dev, err := reg.mdls.Device().FindByMac(ctx, mac)
	if err != nil {
		if err == pgx.ErrNoRows && dev == nil {
			log.Println("no device found by mac, creating new device...")
			// create new device record
			dev, err = reg.mdls.Device().Create(ctx, mac, ip, hostname)
			if err != nil {
				return nil, err
			}

			clnt := NewClientDevice(reg.db, reg.mdls, dev)

			// call createdHooks functions
			if len(reg.createdHooks) > 0 {
				for _, hookFn := range reg.createdHooks {
					if err := hookFn(ctx, clnt); err != nil {
						return nil, err
					}
				}
			}

			return clnt, nil
		}

		log.Println("error finding device by mac:", err)
		return nil, err
	}

	clnt := NewClientDevice(reg.db, reg.mdls, dev)
	changed := ip != dev.IpAddress() || hostname != dev.Hostname()

	// Update device details if need be
	if changed {
		connected := reg.mgr.IsConnected(clnt)
		if connected {
			// disconnect temporarily
			err = reg.mgr.Disconnect(ctx, clnt, "Device details changed, reconnecting...")
			if err != nil {
				return nil, err
			}
		}

		old := NewClientDevice(reg.db, reg.mdls, dev.Clone())
		err := dev.Update(ctx, mac, ip, hostname)
		if err != nil {
			fmt.Println("error updating dev: ", err)
			return nil, fmt.Errorf("could not update dev: %w", err)
		}

		// call changedHooks functions
		if len(reg.changedHooks) > 0 {
			for _, hookFn := range reg.changedHooks {
				if err := hookFn(ctx, clnt, old); err != nil {
					return nil, err
				}
			}
		}

		// reconnect client device
		if connected {
			err := reg.mgr.Connect(ctx, clnt, "Device details changed, reconnected successfully!")
			if err != nil {
				return nil, err
			}
		}
	}

	return clnt, nil
}
