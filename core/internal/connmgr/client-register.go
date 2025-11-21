package connmgr

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"core/db"
	"core/db/models"
	sdkapi "sdk/api"
)

func NewClientRegister(dtb *db.Database, mdls *models.Models) *ClientRegister {
	return &ClientRegister{
		db:   dtb,
		mdls: mdls,
	}
}

type ClientRegister struct {
	db          *db.Database
	mdls        *models.Models
	sessionsMgr *SessionsMgr
}

func (reg *ClientRegister) SetSessionsMgr(mgr *SessionsMgr) {
	reg.sessionsMgr = mgr
}

func (reg *ClientRegister) Register(dtb *db.Database, r *http.Request, mac string, ip string, hostname string) (sdkapi.IClientDevice, error) {
	ctx := r.Context()

	var clnt sdkapi.IClientDevice

	dev, err := reg.mdls.Device().FindByMac(ctx, mac)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && dev == nil {
			log.Println("no device found by mac, creating new device...")
			// create new device record
			dev, err = reg.mdls.Device().Create(ctx, models.CreateDeviceParams{
				MacAddress: mac,
				IpAddress:  ip,
				Hostname:   hostname,
			})
			if err != nil {
				return nil, err
			}

			clnt = NewClientDevice(reg.db, reg.mdls, dev)
			reg.sessionsMgr.emitClientEvent(sdkapi.EventClientCreated, clnt)

			return clnt, nil
		}

		log.Println("error finding device by mac:", err)
		return nil, err
	}

	clnt = NewClientDevice(reg.db, reg.mdls, dev)
	changed := ip != dev.IpAddr() || hostname != dev.Hostname()

	// Update device details if need be
	if changed {
		connected := reg.sessionsMgr.IsConnected(clnt)
		if connected {
			// disconnect temporarily
			err = reg.sessionsMgr.Disconnect(ctx, clnt, "Device details changed, reconnecting...")
			if err != nil {
				return nil, err
			}
		}

		// old := NewClientDevice(reg.db, reg.mdls, dev.Clone())
		// Devices are have disconnected status by default.
		err := dev.Update(ctx, models.UpdateDeviceParams{
			ID:         dev.Id(),
			MacAddress: mac,
			IpAddress:  ip,
			Hostname:   hostname,
			Status:     int(sdkapi.Disconnected),
		})
		if err != nil {
			fmt.Println("error updating dev to connected: ", err)
			return nil, fmt.Errorf("could not update dev to connected: %w", err)
		}

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientUpdated, clnt)

		// reconnect client device
		if connected {
			err := reg.sessionsMgr.Connect(ctx, clnt, "Device details changed, reconnected successfully!")
			if err != nil {
				return nil, err
			}

			if err := dev.Update(ctx, models.UpdateDeviceParams{
				ID:         dev.Id(),
				MacAddress: mac,
				IpAddress:  ip,
				Hostname:   hostname,
				Status:     int(sdkapi.Connected),
			}); err != nil {
				fmt.Println("error updating dev to connected: ", err)
				return nil, fmt.Errorf("could not update dev to connected: %w", err)
			}
		}

		if err := dev.Update(ctx, models.UpdateDeviceParams{
			ID:         dev.Id(),
			MacAddress: mac,
			IpAddress:  ip,
			Hostname:   hostname,
			Status:     int(sdkapi.Connected),
		}); err != nil {
			fmt.Println("error updating dev to connected: ", err)
			return nil, fmt.Errorf("could not update dev to connected: %w", err)
		}
	}

	return clnt, nil
}
