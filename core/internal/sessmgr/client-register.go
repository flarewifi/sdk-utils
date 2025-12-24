package sessmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

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
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Updating device - DeviceID=%d, OldMAC=%s, NewMAC=%s, OldIP=%s, NewIP=%s, OldHostname=%s, NewHostname=%s",
		clnt.ID(), clnt.MacAddr(), newMac, clnt.IpAddr(), newIP, clnt.Hostname(), newHostname)

	// Check if device has a running session (before updating)
	rs, hasRunningSession := reg.sessionsMgr.GetRunningSession(clnt)
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Device has running session: %v", hasRunningSession)

	// If there's a running session, update its network details first
	// This ensures TC rules and internal state are updated before disconnect/reconnect
	if hasRunningSession {
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Updating running session network details")
		// Update the running session's MAC and IP addresses
		// This will update TC (traffic control) rules to point to the new IP
		if err := rs.UpdateNetworkDetails(ctx, newMac, newIP); err != nil {
			log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to update running session network details: %v", err)
			return fmt.Errorf("failed to update running session network details: %w", err)
		}

		// Now disconnect the session (it already has updated network details)
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Disconnecting session before update")
		err := reg.sessionsMgr.Disconnect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("info", "Device network details changed, reconnecting"))
		if err != nil {
			log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to disconnect session: %v", err)
			return err
		}
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Session disconnected successfully")
	}

	// Emit before update hook and check for errors
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Emitting EventClientBeforeUpdated")
	if err := reg.sessionsMgr.emitClientEvent(sdkapi.EventClientBeforeUpdated, clnt); err != nil {
		log.Printf("[ClientRegister.UpdateDevice] ERROR: EventClientBeforeUpdated hook failed: %v", err)
		return err
	}

	// Update device in database with new network details
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Updating device in database")
	err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
		Mac:      newMac,
		Ip:       newIP,
		Hostname: newHostname,
		UUID:     clnt.UUID(),              // Preserve existing UUID
		Status:   int(sdkapi.Disconnected), // Set to disconnected during update
	})
	if err != nil {
		log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to update device in database: %v", err)
		return fmt.Errorf("could not update device: %w", err)
	}
	log.Printf("[ClientRegister.UpdateDevice] SUCCESS: Device updated in database")

	reg.sessionsMgr.emitClientEvent(sdkapi.EventClientUpdated, clnt)
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Emitted EventClientUpdated")

	// Reconnect if was previously running a session
	if hasRunningSession {
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Reconnecting session")
		err := reg.sessionsMgr.Connect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("success", "Device network details updated, reconnected successfully"))
		if err != nil {
			log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to reconnect session: %v", err)
			return err
		}

		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Updating device status to connected")
		if err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
			Mac:      newMac,
			Ip:       newIP,
			Hostname: newHostname,
			UUID:     clnt.UUID(), // Preserve existing UUID
			Status:   int(sdkapi.Connected),
		}); err != nil {
			log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to update device to connected: %v", err)
			return fmt.Errorf("could not update device to connected: %w", err)
		}
		log.Printf("[ClientRegister.UpdateDevice] SUCCESS: Session reconnected successfully")
	}

	log.Printf("[ClientRegister.UpdateDevice] SUCCESS: UpdateDevice completed for DeviceID=%d", clnt.ID())
	return nil
}

type ClientRegisterParams struct {
	CookieDeviceID *int64
	MacAddr        string
	IpAddr         string
	Hostname       string
}

func (reg *ClientRegister) Register(ctx context.Context, params ClientRegisterParams) (sdkapi.IClientDevice, bool, error) {
	log.Printf("[ClientRegister] DEBUG: Register called - CookieDeviceID=%v, MAC=%s, IP=%s, Hostname=%s",
		params.CookieDeviceID, params.MacAddr, params.IpAddr, params.Hostname)

	var clnt sdkapi.IClientDevice

	// Step 1: If cookie exists, prioritize it (cookie identifies the user/device)
	if params.CookieDeviceID != nil {
		log.Printf("[ClientRegister] DEBUG: Step 1 - Cookie provided (ID=%d), looking up device", *params.CookieDeviceID)
		clnt, err := reg.FindByID(ctx, *params.CookieDeviceID)
		if err == nil && clnt != nil {
			log.Printf("[ClientRegister] DEBUG: Found device by cookie - DeviceID=%d, CurrentMAC=%s, CurrentIP=%s",
				clnt.ID(), clnt.MacAddr(), clnt.IpAddr())

			// Check if MAC address changed
			if clnt.MacAddr() != params.MacAddr {
				log.Printf("[ClientRegister] WARN: MAC address changed - Old=%s, New=%s", clnt.MacAddr(), params.MacAddr)

				// MAC changed - check if new MAC already belongs to another device
				existingDev, macErr := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
				if macErr == nil && existingDev != nil && existingDev.ID() != clnt.ID() {
					// MAC conflict: New MAC already belongs to another device
					// This prevents cookie sharing across devices
					log.Printf("[ClientRegister] WARN: MAC conflict detected - New MAC %s belongs to DeviceID=%d, keeping cookie device ID=%d",
						params.MacAddr, existingDev.ID(), clnt.ID())

					ipChanged := params.IpAddr != clnt.IpAddr()
					hostnameChanged := params.Hostname != clnt.Hostname()

					if ipChanged || hostnameChanged {
						log.Printf("[ClientRegister] DEBUG: Updating IP/Hostname only (MAC conflict) - IP changed=%v, Hostname changed=%v",
							ipChanged, hostnameChanged)
						err := reg.UpdateDevice(ctx, clnt, clnt.MacAddr(), params.IpAddr, params.Hostname)
						if err != nil {
							log.Printf("[ClientRegister] ERROR: Failed to update device: %v", err)
							return nil, false, err
						}
					}
					log.Printf("[ClientRegister] SUCCESS: Returned existing device (MAC conflict handled) - DeviceID=%d", clnt.ID())
					return clnt, true, nil
				}

				// No conflict - legitimate MAC change (randomization, new adapter, etc.)
				log.Printf("[ClientRegister] DEBUG: No MAC conflict, updating device with new MAC")
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to update device with new MAC: %v", err)
					return nil, false, err
				}
				log.Printf("[ClientRegister] SUCCESS: Updated device with new MAC - DeviceID=%d", clnt.ID())
				return clnt, true, nil
			}

			// MAC hasn't changed, check if IP or hostname changed
			ipChanged := params.IpAddr != clnt.IpAddr()
			hostnameChanged := params.Hostname != clnt.Hostname()

			if ipChanged || hostnameChanged {
				log.Printf("[ClientRegister] DEBUG: Network details changed - IP changed=%v, Hostname changed=%v",
					ipChanged, hostnameChanged)
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to update network details: %v", err)
					return nil, false, err
				}
				log.Printf("[ClientRegister] SUCCESS: Updated network details - DeviceID=%d", clnt.ID())
			} else {
				log.Printf("[ClientRegister] DEBUG: No changes detected, returning existing device - DeviceID=%d", clnt.ID())
			}

			return clnt, true, nil
		} else if err != nil {
			log.Printf("[ClientRegister] WARN: Failed to find device by cookie ID=%d: %v", *params.CookieDeviceID, err)
		}
	}

	// Step 2: No valid cookie - try to find device by MAC
	log.Printf("[ClientRegister] DEBUG: Step 2 - No valid cookie, searching by MAC=%s", params.MacAddr)
	dev, err := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
	if err == nil && dev != nil {
		log.Printf("[ClientRegister] DEBUG: Found device by MAC - DeviceID=%d, IP=%s, Hostname=%s",
			dev.ID(), dev.IpAddr(), dev.Hostname())
		clnt = NewClientDevice(reg.db, reg.mdls, dev)

		// Check if IP or hostname changed
		ipChanged := params.IpAddr != dev.IpAddr()
		hostnameChanged := params.Hostname != dev.Hostname()

		if ipChanged || hostnameChanged {
			log.Printf("[ClientRegister] DEBUG: Network details changed - IP changed=%v, Hostname changed=%v",
				ipChanged, hostnameChanged)
			err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
			if err != nil {
				log.Printf("[ClientRegister] ERROR: Failed to update device: %v", err)
				return nil, false, err
			}
			log.Printf("[ClientRegister] SUCCESS: Updated device found by MAC - DeviceID=%d", dev.ID())
		} else {
			log.Printf("[ClientRegister] DEBUG: No changes, returning existing device - DeviceID=%d", dev.ID())
		}

		return clnt, true, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] ERROR: Database error searching by MAC=%s: %v", params.MacAddr, err)
	}

	// Step 3: Not found by cookie or MAC - create new device
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] DEBUG: Step 3 - Device not found, creating new device - MAC=%s, IP=%s, Hostname=%s",
			params.MacAddr, params.IpAddr, params.Hostname)
		dev, err = reg.mdls.Device().Create(ctx, models.CreateDeviceParams{
			MacAddress: params.MacAddr,
			IpAddress:  params.IpAddr,
			Hostname:   params.Hostname,
		})
		if err != nil {
			log.Printf("[ClientRegister] ERROR: Failed to create device: %v", err)
			return nil, false, err
		}

		log.Printf("[ClientRegister] SUCCESS: Created new device - DeviceID=%d, MAC=%s, IP=%s",
			dev.ID(), params.MacAddr, params.IpAddr)
		clnt = NewClientDevice(reg.db, reg.mdls, dev)

		// Emit before created hook and check for errors
		if err := reg.sessionsMgr.emitClientEvent(sdkapi.EventClientBeforeCreated, clnt); err != nil {
			log.Printf("[ClientRegister] ERROR: EventClientBeforeCreated hook failed: %v", err)
			return nil, false, err
		}

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientCreated, clnt)
		log.Printf("[ClientRegister] DEBUG: Emitted EventClientCreated for DeviceID=%d", dev.ID())

		return clnt, true, nil
	}

	log.Printf("[ClientRegister] ERROR: Unexpected error: %v", err)
	return nil, false, err
}
