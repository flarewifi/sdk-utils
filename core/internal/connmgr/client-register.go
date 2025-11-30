package connmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

// FindByID finds a client device by its database ID
func (reg *ClientRegister) FindByID(ctx context.Context, deviceID int64) (sdkapi.IClientDevice, error) {
	dev, err := reg.mdls.Device().Find(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	clnt := NewClientDevice(reg.db, reg.mdls, dev)
	return clnt, nil
}

// UpdateDevice updates device network details and handles reconnection if needed
func (reg *ClientRegister) UpdateDevice(ctx context.Context, clnt sdkapi.IClientDevice, newMac, newIP, newHostname string) error {
	// Check if device has a running session (before updating)
	_, hasRunningSession := reg.sessionsMgr.GetRunningSession(clnt)

	// Disconnect if currently has a running session
	if hasRunningSession {
		err := reg.sessionsMgr.Disconnect(ctx, clnt, "Device network details changed, reconnecting...")
		if err != nil {
			return err
		}
	}

	// Update device in database with new network details
	err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
		Mac:      newMac,
		Ip:       newIP,
		Hostname: newHostname,
		Status:   int(sdkapi.Disconnected), // Set to disconnected during update
	})
	if err != nil {
		return fmt.Errorf("could not update device: %w", err)
	}

	reg.sessionsMgr.emitClientEvent(sdkapi.EventClientUpdated, clnt)

	// Reconnect if was previously running a session
	if hasRunningSession {
		err := reg.sessionsMgr.Connect(ctx, clnt, "Device network details updated, reconnected successfully!")
		if err != nil {
			return err
		}

		if err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
			Mac:      newMac,
			Ip:       newIP,
			Hostname: newHostname,
			Status:   int(sdkapi.Connected),
		}); err != nil {
			return fmt.Errorf("could not update device to connected: %w", err)
		}
	}

	return nil
}

type ClientRegisterParams struct {
	CookieDeviceID *int64
	MacAddr        string
	IpAddr         string
	Hostname       string
}

func (reg *ClientRegister) Register(ctx context.Context, params ClientRegisterParams) (sdkapi.IClientDevice, bool, error) {
	var clnt sdkapi.IClientDevice

	// Step 1: If cookie exists, prioritize it (cookie identifies the user/device)
	if params.CookieDeviceID != nil {
		clnt, err := reg.FindByID(ctx, *params.CookieDeviceID)
		if err == nil && clnt != nil {
			// Check if MAC address changed
			if clnt.MacAddr() != params.MacAddr {
				// MAC changed - check if new MAC already belongs to another device
				existingDev, macErr := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
				if macErr == nil && existingDev != nil && existingDev.ID() != clnt.ID() {
					// MAC conflict: New MAC already belongs to another device
					// This prevents cookie sharing across devices
					// Keep the cookie device, but only update IP/hostname (not MAC)
					ipChanged := params.IpAddr != clnt.IpAddr()
					hostnameChanged := params.Hostname != clnt.Hostname()

					if ipChanged || hostnameChanged {
						err := reg.UpdateDevice(ctx, clnt, clnt.MacAddr(), params.IpAddr, params.Hostname)
						if err != nil {
							return nil, false, err
						}
					}
					return clnt, true, nil
				}

				// No conflict - legitimate MAC change (randomization, new adapter, etc.)
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					return nil, false, err
				}
				return clnt, true, nil
			}

			// MAC hasn't changed, check if IP or hostname changed
			ipChanged := params.IpAddr != clnt.IpAddr()
			hostnameChanged := params.Hostname != clnt.Hostname()

			if ipChanged || hostnameChanged {
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					return nil, false, err
				}
			}

			return clnt, true, nil
		}
	}

	// Step 2: No valid cookie - try to find device by MAC
	dev, err := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
	if err == nil && dev != nil {
		clnt = NewClientDevice(reg.db, reg.mdls, dev)

		// Check if IP or hostname changed
		ipChanged := params.IpAddr != dev.IpAddr()
		hostnameChanged := params.Hostname != dev.Hostname()

		if ipChanged || hostnameChanged {
			err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
			if err != nil {
				return nil, false, err
			}
		}

		return clnt, true, nil
	}

	// Step 3: Not found by cookie or MAC - create new device
	if errors.Is(err, sql.ErrNoRows) {
		dev, err = reg.mdls.Device().Create(ctx, models.CreateDeviceParams{
			MacAddress: params.MacAddr,
			IpAddress:  params.IpAddr,
			Hostname:   params.Hostname,
		})
		if err != nil {
			return nil, false, err
		}

		clnt = NewClientDevice(reg.db, reg.mdls, dev)
		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientCreated, clnt)

		return clnt, true, nil
	}

	return nil, false, err
}
