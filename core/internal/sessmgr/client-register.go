package sessmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"core/db"
	"core/db/models"
	"core/db/queries"
	browserdetect "core/internal/modules/browser-detect"
	"core/internal/modules/fingerprint"
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

// wrapDevice wraps a models.Device into a ClientDevice with IsConnected callback.
func (reg *ClientRegister) wrapDevice(d *models.Device) *ClientDevice {
	clnt := NewClientDevice(reg.db, reg.mdls, d)
	if reg.sessionsMgr != nil {
		clnt.SetIsConnectedFunc(reg.sessionsMgr.isDeviceConnected)
	}
	return clnt
}

// FindByID finds a client device by its database ID
func (reg *ClientRegister) FindByID(ctx context.Context, deviceID int64) (sdkapi.IClientDevice, error) {
	dev, err := reg.mdls.Device().Find(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return reg.wrapDevice(dev), nil
}

// UpdateDevice updates device network details and handles reconnection if needed.
// If the new MAC address belongs to another device, this function will merge the devices
// (transferring sessions, purchases, fingerprints, and wallet balance) since fingerprint
// validation has already passed before this function is called.
func (reg *ClientRegister) UpdateDevice(ctx context.Context, clnt sdkapi.IClientDevice, newMac, newIP, newHostname string) error {
	// Check for MAC collision - if new MAC belongs to another device, we need to merge
	// This happens when MAC randomization creates a new device record, but fingerprint
	// validation proves it's the same physical device
	if clnt.MacAddr() != newMac {
		existingDev, err := reg.mdls.Device().FindByMac(ctx, newMac)
		if err == nil && existingDev != nil && existingDev.ID() != clnt.ID() {
			// MAC collision detected - DeviceID wants newMac but another device already has it
			// Check if the conflicting device has an active session
			conflictingClnt := reg.wrapDevice(existingDev)
			if _, hasSession := reg.sessionsMgr.GetRunningSession(conflictingClnt); hasSession {
				// Disconnect active session on conflicting device before merge
				if err := reg.sessionsMgr.Disconnect(ctx, conflictingClnt, ""); err != nil {
					log.Printf("[ClientRegister.UpdateDevice] WARN: Failed to disconnect conflicting device session: %v", err)
					// Continue with merge anyway
				}
			}

			// Merge the conflicting device into the current device
			// This transfers all sessions, purchases, fingerprints, and wallet balance
			if err := reg.mdls.Device().MergeDevices(ctx, clnt.ID(), existingDev.ID()); err != nil {
				return fmt.Errorf("could not merge devices: %w", err)
			}
		}
	}

	// Check if device has a running session
	_, hasRunningSession := reg.sessionsMgr.GetRunningSession(clnt)

	// Disconnect if running (this handles TC cleanup, nftables, etc.)
	if hasRunningSession {
		err := reg.sessionsMgr.Disconnect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("info", "Device network details changed, reconnecting"))
		if err != nil {
			return err
		}
	}

	// Update device in database with new network details
	err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
		Mac:      newMac,
		Ip:       newIP,
		Hostname: newHostname,
		UUID:     clnt.UUID(),                     // Preserve existing UUID
		Status:   sdkapi.DeviceStatusDisconnected, // Set to disconnected during update
	})
	if err != nil {
		return fmt.Errorf("could not update device: %w", err)
	}

	reg.sessionsMgr.emitClientEvent(sdkapi.EventClientUpdated, clnt)

	// Reconnect if was previously running a session
	if hasRunningSession {
		err := reg.sessionsMgr.Connect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("success", "Device network details updated, reconnected successfully"))
		if err != nil {
			return err
		}

		// Update device status to connected
		if err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
			Mac:      newMac,
			Ip:       newIP,
			Hostname: newHostname,
			UUID:     clnt.UUID(),
			Status:   sdkapi.DeviceStatusConnected,
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
	// Fingerprint data
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

// Register registers or identifies a device based on cookie, MAC address, or creates a new device.
// It validates device fingerprints to prevent cookie sharing and MAC collision attacks.
// Returns (device, shouldSetCookie, error)
func (reg *ClientRegister) Register(ctx context.Context, params ClientRegisterParams) (sdkapi.IClientDevice, bool, error) {
	// Parse browser info and generate fingerprint hash
	browserInfo := browserdetect.DetectBrowser(params.UserAgent)
	fpHash := ""

	// Accept fingerprint data even if partial (for CNA support) or minimal (JS disabled)
	// Full fingerprint: UserAgent + ScreenRes + Language + Timezone
	// Partial fingerprint (CNA): UserAgent only (ScreenRes/Language/Timezone empty), IsCNA=true
	// Minimal fingerprint (JS disabled): UserAgent only, IsCNA=false
	hasFullFingerprintData := params.UserAgent != "" && params.ScreenRes != "" && params.Language != ""
	hasCNAFingerprintData := params.UserAgent != "" && browserInfo.IsCNA
	hasMinimalFingerprintData := params.UserAgent != "" && !hasFullFingerprintData && !hasCNAFingerprintData
	hasFingerprintData := hasFullFingerprintData || hasCNAFingerprintData || hasMinimalFingerprintData

	if hasFingerprintData {
		fpData := fingerprint.FingerprintData{
			UserAgent: params.UserAgent,
			ScreenRes: params.ScreenRes,
			Language:  params.Language,
			Timezone:  params.Timezone,
		}
		fpHash = fingerprint.GenerateHash(fpData)
	}

	var clnt sdkapi.IClientDevice

	// Step 1: If cookie exists, prioritize it (cookie identifies the user/device)
	if params.CookieDeviceID != nil {
		clnt, err := reg.FindByID(ctx, *params.CookieDeviceID)
		if err == nil && clnt != nil {
			// Validate fingerprint if we have it
			if hasFingerprintData {
				isValid, matchedFP, err := reg.validateDeviceFingerprint(ctx, clnt.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)

				if err != nil {
					// Fingerprint validation error, fall through to MAC match
					goto STEP_2_MAC_MATCH
				}

				if !isValid {
					// Fingerprint validation failed - possible cookie sharing detected
					goto STEP_2_MAC_MATCH
				}

				// Valid fingerprint - update or add it
				if matchedFP != nil {
					// Exact match found, update last_seen
					reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, matchedFP.ID)
				} else {
					// Smart match or first time - add new fingerprint variant
					reg.addFingerprint(ctx, clnt.ID(), params, browserInfo, fpHash)
				}
			} else {
				// No full fingerprint data - try OS-only validation if we have User-Agent
				if params.UserAgent != "" && browserInfo.OSFamily != "" {
					// Use validateDeviceFingerprint with OS-only mode (empty hash/screen/lang, not CNA)
					isValid, _, err := reg.validateDeviceFingerprint(ctx, clnt.ID(), "", "", browserInfo.OSFamily, "", "", false)
					if err != nil {
						goto STEP_2_MAC_MATCH
					}
					if !isValid {
						// OS family doesn't match any stored fingerprint - possible cookie theft
						goto STEP_2_MAC_MATCH
					}
					// OS matches or no stored fingerprints - accept cookie (minimal validation passed)
				} else {
					// No User-Agent - check if device has stored fingerprints
					storedFingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, clnt.ID())
					if err == nil && len(storedFingerprints) > 0 {
						// Device has fingerprints but we can't validate - reject for security
						goto STEP_2_MAC_MATCH
					}
					// No stored fingerprints - accept (backward compatibility)
				}
			}

			// Update network details if changed
			if clnt.MacAddr() != params.MacAddr || clnt.IpAddr() != params.IpAddr || clnt.Hostname() != params.Hostname {
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					return nil, false, err
				}
			}

			reg.sessionsMgr.emitClientEvent(sdkapi.EventClientRegistered, clnt)
			return clnt, true, nil
		}
		// Failed to find device by cookie, fall through to MAC match
	}

STEP_2_MAC_MATCH:
	// Step 2: No valid cookie - try to find device by current MAC
	dev, err := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
	if err == nil && dev != nil {
		clnt = reg.wrapDevice(dev)

		// Validate fingerprint if we have it
		if hasFingerprintData {
			isValid, matchedFP, err := reg.validateDeviceFingerprint(ctx, dev.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)

			if err != nil {
				// Fingerprint validation error, try MAC history fallback
				dev = nil
				goto STEP_2_5_MAC_HISTORY
			}

			if !isValid {
				// Fingerprint validation failed - possible MAC collision
				dev = nil
				goto STEP_2_5_MAC_HISTORY
			}

			// Valid fingerprint - update or add it
			if matchedFP != nil {
				reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, matchedFP.ID)
			} else {
				reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash)
			}
		} else {
			// No full fingerprint data - try OS-only validation if we have User-Agent
			if params.UserAgent != "" && browserInfo.OSFamily != "" {
				// Use validateDeviceFingerprint with OS-only mode (empty hash/screen/lang, not CNA)
				isValid, _, err := reg.validateDeviceFingerprint(ctx, dev.ID(), "", "", browserInfo.OSFamily, "", "", false)
				if err != nil {
					dev = nil
					goto STEP_2_5_MAC_HISTORY
				}
				if !isValid {
					// OS family doesn't match any stored fingerprint - possible MAC collision
					dev = nil
					goto STEP_2_5_MAC_HISTORY
				}
				// OS matches or no stored fingerprints - accept MAC match (minimal validation passed)
			} else {
				// No User-Agent - check if device has stored fingerprints
				storedFingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, dev.ID())
				if err == nil && len(storedFingerprints) > 0 {
					// Device has fingerprints but we can't validate - reject for security
					dev = nil
					goto STEP_2_5_MAC_HISTORY
				}
				// No stored fingerprints - accept (backward compatibility)
			}
		}

		// Update network details if changed (MAC change will be recorded by UpdateDevice)
		if dev.IpAddr() != params.IpAddr || dev.Hostname() != params.Hostname || dev.MacAddr() != params.MacAddr {
			err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
			if err != nil {
				return nil, false, err
			}
		}

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientRegistered, clnt)
		return clnt, true, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] ERROR: Database error searching by MAC: %v", err)
	}

STEP_2_5_MAC_HISTORY:
	// Step 2.5: Try to find device by MAC history (handles MAC randomization when device returns with old MAC)
	deviceID, err := reg.mdls.DeviceMac().FindDeviceByAnyMac(ctx, params.MacAddr)
	if err == nil && deviceID > 0 {
		// Load the device
		dev, err = reg.mdls.Device().Find(ctx, deviceID)
		if err != nil {
			goto STEP_3_CREATE_NEW
		}

		clnt = reg.wrapDevice(dev)

		// CRITICAL: Must validate fingerprint since this MAC might have been reused by different device
		if !hasFingerprintData {
			// No fingerprint data to validate - reject for security
			dev = nil
			goto STEP_3_CREATE_NEW
		}

		isValid, matchedFP, err := reg.validateDeviceFingerprint(ctx, dev.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)

		if err != nil {
			dev = nil
			goto STEP_3_CREATE_NEW
		}

		if !isValid {
			// Fingerprint validation failed - MAC was reused by different device
			dev = nil
			goto STEP_3_CREATE_NEW
		}

		// Valid fingerprint - update or add it
		if matchedFP != nil {
			reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, matchedFP.ID)
		} else {
			reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash)
		}

		// Update device to use this old MAC as current (MAC rotation back)
		err = reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
		if err != nil {
			return nil, false, err
		}

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientRegistered, clnt)
		return clnt, true, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] ERROR: Database error searching MAC history: %v", err)
	}

STEP_3_CREATE_NEW:
	// Step 3: Not found by cookie or MAC - create new device
	if errors.Is(err, sql.ErrNoRows) || dev == nil {
		dev, err = reg.mdls.Device().Create(ctx, models.CreateDeviceParams{
			MacAddress: params.MacAddr,
			IpAddress:  params.IpAddr,
			Hostname:   params.Hostname,
		})
		if err != nil {
			return nil, false, err
		}

		clnt = reg.wrapDevice(dev)

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientCreated, clnt)
		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientRegistered, clnt)

		// Add first fingerprint for new device (full, partial/CNA, or minimal/JS-disabled)
		if hasFingerprintData && fpHash != "" {
			reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash)
		}

		return clnt, true, nil
	}

	return nil, false, err
}

// validateDeviceFingerprint checks if current fingerprint matches any stored fingerprints.
// Supports three validation modes:
// 1. Full fingerprint: Hash + Screen + Lang + TZ + OS
// 2. Partial fingerprint (CNA): OS only with IsCNA flag
// 3. Minimal fingerprint (OS-only): When only User-Agent/OS is available (e.g., JS disabled)
// Returns (isValid, matchedFingerprint, error)
func (reg *ClientRegister) validateDeviceFingerprint(ctx context.Context, deviceID int64, currentHash string, currentScreen string, currentOS string, currentLang string, currentTZ string, currentIsCNA bool) (bool, *queries.DeviceFingerprint, error) {
	// Get all fingerprints for device (within 6 months)
	fingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, deviceID)
	if err != nil {
		return false, nil, err
	}

	// No stored fingerprints - first time registration, accept
	if len(fingerprints) == 0 {
		return true, nil, nil
	}

	// Determine if this is an OS-only validation (minimal fingerprint)
	// This happens when we have User-Agent (OS) but no other fingerprint data
	isOSOnlyValidation := currentOS != "" && currentHash == "" && currentScreen == "" && currentLang == "" && !currentIsCNA

	// Check against all stored fingerprints
	for i := range fingerprints {
		fp := &fingerprints[i]

		// OS-only validation mode: just check if OS family matches
		// This is used when device has cookie/token but JS is disabled or fingerprint collection failed
		if isOSOnlyValidation {
			if fp.OsFamily != "" && fp.OsFamily == currentOS {
				// OS matches - minimal validation passed
				// Return nil for matchedFP since we don't have full fingerprint to update
				return true, nil, nil
			}
			continue
		}

		// Full/partial fingerprint validation
		result := fingerprint.ValidateFingerprint(
			fingerprint.StoredFingerprint{
				FingerprintHash:  fp.FingerprintHash,
				OSFamily:         fp.OsFamily,
				ScreenResolution: fp.ScreenResolution,
				Language:         fp.Language,
				Timezone:         fp.Timezone,
				IsCna:            fp.IsCna,
			},
			currentHash,
			currentOS,
			currentScreen,
			currentLang,
			currentTZ,
			currentIsCNA,
		)

		if result == fingerprint.ExactMatch {
			return true, fp, nil
		}

		if result == fingerprint.SmartMatch {
			// Smart match scenarios:
			// - CNA-to-CNA match (same OS)
			// - CNA fingerprint accepting browser (same OS)
			// - Browser fingerprint accepting CNA (same OS)
			// - Same OS+Screen+Lang+TZ but different hash (browser switch detected)
			return true, nil, nil
		}
	}

	// No match found - different device
	return false, nil, nil
}

// addFingerprint creates a new fingerprint record for device
func (reg *ClientRegister) addFingerprint(ctx context.Context, deviceID int64, params ClientRegisterParams, browserInfo browserdetect.BrowserInfo, fpHash string) error {
	// Don't store empty fingerprint hashes
	if fpHash == "" {
		return fmt.Errorf("fingerprint hash is empty")
	}

	// Check if exact fingerprint already exists
	existing, err := reg.mdls.DeviceFingerprint().CheckExactMatch(ctx, deviceID, fpHash)
	if err != nil {
		log.Printf("[Fingerprint] ERROR: Failed to check for existing fingerprint: %v", err)
	}

	if existing != nil {
		// Fingerprint already exists, update last_seen timestamp
		return reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, existing.ID)
	}

	// Check fingerprint limit (max 10 per device to prevent flooding)
	fingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, deviceID)
	if err != nil {
		// Continue anyway - don't block on this check
	} else if len(fingerprints) >= 10 {
		return fmt.Errorf("fingerprint limit reached (max 10 per device)")
	}

	// Create new fingerprint record
	_, err = reg.mdls.DeviceFingerprint().Create(ctx, queries.CreateDeviceFingerprintParams{
		DeviceID:         deviceID,
		FingerprintHash:  fpHash,
		UserAgent:        params.UserAgent,
		BrowserName:      browserInfo.BrowserName,
		OsFamily:         browserInfo.OSFamily,
		ScreenResolution: params.ScreenRes,
		Language:         params.Language,
		Timezone:         params.Timezone,
		IsCna:            browserInfo.IsCNA,
	})

	return err
}
