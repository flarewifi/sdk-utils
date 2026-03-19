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
	"core/utils/arp"
	"core/utils/ndp"
	sdkapi "sdk/api"
)

// =============================================================================
// TYPES
// =============================================================================

type ClientRegister struct {
	db          *db.Database
	mdls        *models.Models
	sessionsMgr *SessionsMgr
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

func NewClientRegister(dtb *db.Database, mdls *models.Models) *ClientRegister {
	return &ClientRegister{
		db:   dtb,
		mdls: mdls,
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

func (reg *ClientRegister) SetSessionsMgr(mgr *SessionsMgr) {
	reg.sessionsMgr = mgr
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
func (reg *ClientRegister) UpdateDevice(ctx context.Context, clnt sdkapi.IClientDevice, newMac, newIpv4, newIpv6, newHostname string) error {
	// Check for MAC collision - if new MAC belongs to another device (current or historical), we need to merge
	// This happens when MAC randomization creates a new device record, but fingerprint
	// validation proves it's the same physical device
	if clnt.MacAddr() != newMac {
		// Use FindDeviceByAnyMac to check both current and historical MAC ownership
		existingDevID, err := reg.mdls.DeviceMac().FindDeviceByAnyMac(ctx, newMac)
		if err == nil && existingDevID > 0 && existingDevID != clnt.ID() {
			// MAC collision detected - load the existing device
			existingDev, err := reg.mdls.Device().Find(ctx, existingDevID)
			if err != nil {
				return fmt.Errorf("could not find conflicting device %d: %w", existingDevID, err)
			}

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
			sourceDeviceID := existingDev.ID()
			sourceDeviceUUID := existingDev.UUID()
			if err := reg.mdls.Device().MergeDevices(ctx, clnt.ID(), sourceDeviceID); err != nil {
				return fmt.Errorf("could not merge devices: %w", err)
			}

			reg.sessionsMgr.EmitClientMerge(sdkapi.EventClientMergeData{
				Target:           clnt,
				SourceDeviceID:   sourceDeviceID,
				SourceDeviceUUID: sourceDeviceUUID,
			})

			// Note: We don't disconnect the current device here. The regular UpdateDevice flow below
			// (lines 112-154) will handle the disconnect-update-reconnect sequence properly, preserving
			// any active session on the current (target) device.
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
		Ipv4:     newIpv4,
		Ipv6:     newIpv6,
		Hostname: newHostname,
		UUID:     clnt.UUID(),                     // Preserve existing UUID
		Status:   sdkapi.DeviceStatusDisconnected, // Set to disconnected during update
	})
	if err != nil {
		return fmt.Errorf("could not update device: %w", err)
	}

	reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientUpdated, clnt)

	// Reconnect if was previously running a session
	if hasRunningSession {
		err := reg.sessionsMgr.Connect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("success", "Device network details updated, reconnected successfully"))
		if err != nil {
			return err
		}

		// Update device status to connected
		if err := clnt.Update(ctx, sdkapi.UpdateDeviceParams{
			Mac:      newMac,
			Ipv4:     newIpv4,
			Ipv6:     newIpv6,
			Hostname: newHostname,
			UUID:     clnt.UUID(),
			Status:   sdkapi.DeviceStatusConnected,
		}); err != nil {
			return fmt.Errorf("could not update device to connected: %w", err)
		}
	}

	return nil
}

// Register registers or identifies a device based on cookie, MAC address, or creates a new device.
// It validates device fingerprints to prevent cookie sharing and MAC collision attacks.
// Returns (device, shouldSetCookie, error)
func (reg *ClientRegister) Register(ctx context.Context, params ClientRegisterParams) (sdkapi.IClientDevice, bool, error) {
	// Discover both IPv4 and IPv6 addresses for this MAC.
	// The caller may have supplied only one (the one the HTTP request arrived on);
	// we use ARP/NDP to fill in the other so both are stored and connected.
	params.Ipv4Addr, params.Ipv6Addr = discoverBothIPs(params.MacAddr, params.Ipv4Addr, params.Ipv6Addr)

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
			if clnt.MacAddr() != params.MacAddr || clnt.Ipv4Addr() != params.Ipv4Addr || clnt.Ipv6Addr() != params.Ipv6Addr || clnt.Hostname() != params.Hostname {
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.Ipv4Addr, params.Ipv6Addr, params.Hostname)
				if err != nil {
					return nil, false, err
				}
			}

			reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientRegistered, clnt)
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
		if dev.Ipv4Addr() != params.Ipv4Addr || dev.Ipv6Addr() != params.Ipv6Addr || dev.Hostname() != params.Hostname || dev.MacAddr() != params.MacAddr {
			err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.Ipv4Addr, params.Ipv6Addr, params.Hostname)
			if err != nil {
				return nil, false, err
			}
		}

		reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientRegistered, clnt)
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
		err = reg.UpdateDevice(ctx, clnt, params.MacAddr, params.Ipv4Addr, params.Ipv6Addr, params.Hostname)
		if err != nil {
			return nil, false, err
		}

		reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientRegistered, clnt)
		return clnt, true, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] ERROR: Database error searching MAC history: %v", err)
	}

STEP_3_CREATE_NEW:
	// Step 3: Not found by cookie or MAC - check for MAC collision before creating new device
	// This handles cases where fingerprint validation failed but the MAC belongs to another device
	if clnt, shouldSetCookie, handled, handleErr := reg.handleMacCollision(ctx, params, browserInfo, fpHash, hasFingerprintData); handled {
		if handleErr != nil {
			return nil, false, handleErr
		}
		return clnt, shouldSetCookie, nil
	}

	// No MAC collision - create new device
	if errors.Is(err, sql.ErrNoRows) || dev == nil {
		dev, err = reg.mdls.Device().Create(ctx, models.CreateDeviceParams{
			MacAddress:  params.MacAddr,
			Ipv4Address: params.Ipv4Addr,
			Ipv6Address: params.Ipv6Addr,
			Hostname:    params.Hostname,
		})
		if err != nil {
			return nil, false, err
		}

		clnt = reg.wrapDevice(dev)

		reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientCreated, clnt)
		reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientRegistered, clnt)

		// Add first fingerprint for new device (full, partial/CNA, or minimal/JS-disabled)
		if hasFingerprintData && fpHash != "" {
			reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash)
		}

		return clnt, true, nil
	}

	return nil, false, err
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// wrapDevice wraps a models.Device into a ClientDevice.
func (reg *ClientRegister) wrapDevice(d *models.Device) *ClientDevice {
	return NewClientDevice(reg.db, reg.mdls, reg.sessionsMgr, d)
}

// discoverBothIPs looks up both the IPv4 (via ARP) and IPv6 (via NDP) addresses
// for a device identified by its MAC address.  The seed IPs from the HTTP request
// are used as a starting point: if the request IP already tells us one protocol,
// we only do a kernel lookup for the other one.
//
// Parameters:
//
//	mac        – normalized MAC address of the device
//	reqIpv4    – IPv4 from the HTTP request (may be empty if client came via IPv6)
//	reqIpv6    – IPv6 from the HTTP request (may be empty if client came via IPv4)
//
// Returns (ipv4, ipv6) — either or both may be empty.
func discoverBothIPs(mac, reqIpv4, reqIpv6 string) (ipv4, ipv6 string) {
	ipv4 = reqIpv4
	ipv6 = reqIpv6

	// Fill in missing IPv4 from ARP table
	if ipv4 == "" {
		ipv4 = arp.FindIpByMac(mac)
	}

	// Fill in missing IPv6 from NDP table
	if ipv6 == "" {
		ipv6 = ndp.FindIpv6ByMac(mac)
	}

	return ipv4, ipv6
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

// handleMacCollision checks if a MAC address already belongs to an existing device.
// If found and fingerprint validates, it reuses that device instead of creating a new one.
// This prevents duplicate device records for the same MAC address while ensuring security.
// Returns (device, shouldSetCookie, handled, error) where handled=true means caller should return immediately.
func (reg *ClientRegister) handleMacCollision(ctx context.Context, params ClientRegisterParams, browserInfo browserdetect.BrowserInfo, fpHash string, hasFingerprintData bool) (sdkapi.IClientDevice, bool, bool, error) {
	// Check if MAC already belongs to another device
	existingDevID, err := reg.mdls.DeviceMac().FindDeviceByMac(ctx, params.MacAddr)
	if err != nil {
		// No existing device with this MAC - not a collision
		return nil, false, false, nil
	}

	if existingDevID <= 0 {
		// No existing device with this MAC - not a collision
		return nil, false, false, nil
	}

	// MAC collision detected - another device already has this MAC
	log.Printf("[ClientRegister.handleMacCollision] MAC %s already exists on device %d", params.MacAddr, existingDevID)

	existingDev, err := reg.mdls.Device().Find(ctx, existingDevID)
	if err != nil {
		log.Printf("[ClientRegister.handleMacCollision] ERROR: Failed to find existing device %d: %v", existingDevID, err)
		return nil, false, false, nil // Fall through to create new
	}

	// Validate fingerprint before reusing device to prevent identity theft
	if hasFingerprintData && fpHash != "" {
		isValid, _, err := reg.validateDeviceFingerprint(ctx, existingDev.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)
		if err != nil {
			log.Printf("[ClientRegister.handleMacCollision] WARN: fingerprint validation error: %v", err)
			// SECURITY: Cannot create new device - MAC already claimed
			// Return error instead of falling through to device creation
			return nil, false, true, fmt.Errorf("MAC %s already registered but fingerprint validation error: %w", params.MacAddr, err)
		}
		if !isValid {
			log.Printf("[ClientRegister.handleMacCollision] CRITICAL: Fingerprint mismatch for MAC %s on device %d", params.MacAddr, existingDevID)
			// SECURITY: This MAC is already claimed by another device with different fingerprint
			// This could be:
			// 1. MAC address reuse by router (different physical device)
			// 2. Potential security attack (MAC spoofing)
			// 3. Screen rotation issue (should be fixed by normalization, but log for monitoring)
			log.Printf("[ClientRegister.handleMacCollision] Refusing to create duplicate device - MAC %s belongs to device %d", params.MacAddr, existingDevID)
			return nil, false, true, fmt.Errorf("MAC address %s is already registered to a different device", params.MacAddr)
		}
		log.Printf("[ClientRegister.handleMacCollision] Fingerprint validated, reusing device %d", existingDevID)
	} else {
		// No fingerprint data - check if device has stored fingerprints
		storedFPs, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, existingDev.ID())
		if err == nil && len(storedFPs) > 0 {
			// Device has fingerprints but we can't validate - reject for security
			log.Printf("[ClientRegister.handleMacCollision] Device %d has fingerprints but no data to validate", existingDevID)
			// SECURITY: Cannot create new device - MAC already claimed
			return nil, false, true, fmt.Errorf("MAC %s already registered but cannot validate identity", params.MacAddr)
		}
		// No stored fingerprints - accept (backward compatibility)
		log.Printf("[ClientRegister.handleMacCollision] No fingerprints to validate, reusing device %d (backward compatibility)", existingDevID)
	}

	clnt := reg.wrapDevice(existingDev)

	// Update network details if changed
	if existingDev.Ipv4Addr() != params.Ipv4Addr || existingDev.Ipv6Addr() != params.Ipv6Addr || existingDev.Hostname() != params.Hostname {
		err = reg.UpdateDevice(ctx, clnt, params.MacAddr, params.Ipv4Addr, params.Ipv6Addr, params.Hostname)
		if err != nil {
			log.Printf("[ClientRegister.handleMacCollision] ERROR: Failed to update device: %v", err)
			return nil, false, false, err
		}
	}

	// Add fingerprint if we have data (this device may have been created without fingerprints)
	if hasFingerprintData && fpHash != "" {
		reg.addFingerprint(ctx, existingDev.ID(), params, browserInfo, fpHash)
	}

	reg.sessionsMgr.EmitClientEvent(sdkapi.EventClientRegistered, clnt)
	return clnt, true, true, nil
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
