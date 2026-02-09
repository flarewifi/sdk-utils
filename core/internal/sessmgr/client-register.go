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

// FindByID finds a client device by its database ID
func (reg *ClientRegister) FindByID(ctx context.Context, deviceID int64) (sdkapi.IClientDevice, error) {
	dev, err := reg.mdls.Device().Find(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	clnt := NewClientDevice(reg.db, reg.mdls, dev)
	return clnt, nil
}

// UpdateDevice updates device network details and handles reconnection if needed.
// If the new MAC address belongs to another device, this function will merge the devices
// (transferring sessions, purchases, fingerprints, and wallet balance) since fingerprint
// validation has already passed before this function is called.
func (reg *ClientRegister) UpdateDevice(ctx context.Context, clnt sdkapi.IClientDevice, newMac, newIP, newHostname string) error {
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Updating device - DeviceID=%d, OldMAC=%s, NewMAC=%s, OldIP=%s, NewIP=%s, OldHostname=%s, NewHostname=%s",
		clnt.ID(), clnt.MacAddr(), newMac, clnt.IpAddr(), newIP, clnt.Hostname(), newHostname)

	// Check for MAC collision - if new MAC belongs to another device, we need to merge
	// This happens when MAC randomization creates a new device record, but fingerprint
	// validation proves it's the same physical device
	if clnt.MacAddr() != newMac {
		existingDev, err := reg.mdls.Device().FindByMac(ctx, newMac)
		if err == nil && existingDev != nil && existingDev.ID() != clnt.ID() {
			log.Printf("[ClientRegister.UpdateDevice] MAC collision detected - DeviceID=%d wants MAC=%s but DeviceID=%d already has it",
				clnt.ID(), newMac, existingDev.ID())

			// Check if the conflicting device has an active session
			conflictingClnt := NewClientDevice(reg.db, reg.mdls, existingDev)
			if _, hasSession := reg.sessionsMgr.GetRunningSession(conflictingClnt); hasSession {
				log.Printf("[ClientRegister.UpdateDevice] Disconnecting active session on conflicting device %d before merge", existingDev.ID())
				if err := reg.sessionsMgr.Disconnect(ctx, conflictingClnt, ""); err != nil {
					log.Printf("[ClientRegister.UpdateDevice] WARN: Failed to disconnect conflicting device session: %v", err)
					// Continue with merge anyway
				}
			}

			// Merge the conflicting device into the current device
			// This transfers all sessions, purchases, fingerprints, and wallet balance
			log.Printf("[ClientRegister.UpdateDevice] Merging device %d into device %d...", existingDev.ID(), clnt.ID())
			if err := reg.mdls.Device().MergeDevices(ctx, clnt.ID(), existingDev.ID()); err != nil {
				log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to merge devices: %v", err)
				return fmt.Errorf("could not merge devices: %w", err)
			}
			log.Printf("[ClientRegister.UpdateDevice] SUCCESS: Merged device %d into device %d", existingDev.ID(), clnt.ID())
		}
	}

	// Check if device has a running session
	_, hasRunningSession := reg.sessionsMgr.GetRunningSession(clnt)
	log.Printf("[ClientRegister.UpdateDevice] DEBUG: Device has running session: %v", hasRunningSession)

	// Disconnect if running (this handles TC cleanup, nftables, etc.)
	if hasRunningSession {
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Disconnecting session before network update")
		err := reg.sessionsMgr.Disconnect(ctx, clnt, reg.sessionsMgr.coreAPI.Translate("info", "Device network details changed, reconnecting"))
		if err != nil {
			log.Printf("[ClientRegister.UpdateDevice] ERROR: Failed to disconnect session: %v", err)
			return err
		}
		log.Printf("[ClientRegister.UpdateDevice] DEBUG: Session disconnected successfully")
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
	// Fingerprint data
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

func (reg *ClientRegister) Register(ctx context.Context, params ClientRegisterParams) (sdkapi.IClientDevice, bool, error) {
	log.Printf("[ClientRegister] DEBUG: Register called - CookieDeviceID=%v, MAC=%s, IP=%s, Hostname=%s",
		params.CookieDeviceID, params.MacAddr, params.IpAddr, params.Hostname)

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

		if hasMinimalFingerprintData {
			log.Printf("[ClientRegister] DEBUG: Minimal fingerprint generated (JS disabled) - Hash=%s, Browser=%s, OS=%s",
				fpHash[:16]+"...", browserInfo.BrowserName, browserInfo.OSFamily)
		} else if hasCNAFingerprintData && !hasFullFingerprintData {
			log.Printf("[ClientRegister] DEBUG: Partial fingerprint generated (CNA) - Hash=%s, Browser=%s, OS=%s, IsCNA=%v",
				fpHash[:16]+"...", browserInfo.BrowserName, browserInfo.OSFamily, browserInfo.IsCNA)
		} else {
			log.Printf("[ClientRegister] DEBUG: Full fingerprint generated - Hash=%s, Browser=%s, OS=%s, IsCNA=%v",
				fpHash[:16]+"...", browserInfo.BrowserName, browserInfo.OSFamily, browserInfo.IsCNA)
		}
	} else {
		log.Printf("[ClientRegister] WARN: No fingerprint data - UserAgent=%v, ScreenRes=%v, Language=%v, IsCNA=%v",
			params.UserAgent != "", params.ScreenRes != "", params.Language != "", browserInfo.IsCNA)
	}

	var clnt sdkapi.IClientDevice

	// Step 1: If cookie exists, prioritize it (cookie identifies the user/device)
	if params.CookieDeviceID != nil {
		log.Printf("[ClientRegister] DEBUG: Step 1 - Cookie provided (ID=%d), looking up device", *params.CookieDeviceID)
		clnt, err := reg.FindByID(ctx, *params.CookieDeviceID)
		if err == nil && clnt != nil {
			log.Printf("[ClientRegister] DEBUG: Found device by cookie - DeviceID=%d, CurrentMAC=%s, CurrentIP=%s",
				clnt.ID(), clnt.MacAddr(), clnt.IpAddr())

			// Validate fingerprint if we have it
			if hasFingerprintData {
				isValid, matchedFP, err := reg.validateDeviceFingerprint(ctx, clnt.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)

				if err != nil {
					log.Printf("[ClientRegister] ERROR: Fingerprint validation error for DeviceID=%d: %v", clnt.ID(), err)
					goto STEP_2_MAC_MATCH
				}

				if !isValid {
					log.Printf("[ClientRegister] WARN: Fingerprint validation FAILED for cookie DeviceID=%d - possible cookie sharing detected!", clnt.ID())
					log.Printf("[ClientRegister] DEBUG: Rejecting cookie, falling through to MAC match")
					goto STEP_2_MAC_MATCH
				}

				log.Printf("[ClientRegister] DEBUG: Fingerprint validation PASSED for DeviceID=%d", clnt.ID())

				// Valid fingerprint - update or add it
				if matchedFP != nil {
					// Exact match found, update last_seen
					err := reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, matchedFP.ID)
					if err != nil {
						log.Printf("[ClientRegister] ERROR: Failed to update fingerprint last_seen for FingerprintID=%d: %v", matchedFP.ID, err)
					}
				} else {
					// Smart match or first time - add new fingerprint variant
					if err := reg.addFingerprint(ctx, clnt.ID(), params, browserInfo, fpHash); err != nil {
						log.Printf("[ClientRegister] ERROR: Failed to add fingerprint for DeviceID=%d: %v", clnt.ID(), err)
					}
				}
			} else {
				// No fingerprint data provided - check if device has stored fingerprints
				storedFingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, clnt.ID())
				if err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to check stored fingerprints for DeviceID=%d: %v", clnt.ID(), err)
					// Continue anyway - don't block registration on fingerprint errors
				} else if len(storedFingerprints) > 0 {
					// Device has fingerprints but current request doesn't - SUSPICIOUS!
					log.Printf("[ClientRegister] WARN: Device %d has %d stored fingerprint(s) but current request has no fingerprint data - rejecting cookie (possible cookie theft or JavaScript disabled)", clnt.ID(), len(storedFingerprints))
					goto STEP_2_MAC_MATCH
				} else {
					log.Printf("[ClientRegister] DEBUG: Device %d has no stored fingerprints and current request has none - accepting (backward compatibility)", clnt.ID())
				}
			}

			// Update network details if changed
			if clnt.MacAddr() != params.MacAddr || clnt.IpAddr() != params.IpAddr || clnt.Hostname() != params.Hostname {
				log.Printf("[ClientRegister] DEBUG: Network details changed, updating device")
				err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
				if err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to update device: %v", err)
					return nil, false, err
				}
			}

			log.Printf("[ClientRegister] SUCCESS: Returned device from cookie - DeviceID=%d", clnt.ID())
			return clnt, true, nil
		} else if err != nil {
			log.Printf("[ClientRegister] WARN: Failed to find device by cookie ID=%d: %v", *params.CookieDeviceID, err)
		}
	}

STEP_2_MAC_MATCH:
	// Step 2: No valid cookie - try to find device by MAC
	log.Printf("[ClientRegister] DEBUG: Step 2 - Searching by MAC=%s", params.MacAddr)
	dev, err := reg.mdls.Device().FindByMac(ctx, params.MacAddr)
	if err == nil && dev != nil {
		log.Printf("[ClientRegister] DEBUG: Found device by MAC - DeviceID=%d, IP=%s, Hostname=%s",
			dev.ID(), dev.IpAddr(), dev.Hostname())
		clnt = NewClientDevice(reg.db, reg.mdls, dev)

		// Validate fingerprint if we have it
		if hasFingerprintData {
			isValid, matchedFP, err := reg.validateDeviceFingerprint(ctx, dev.ID(), fpHash, params.ScreenRes, browserInfo.OSFamily, params.Language, params.Timezone, browserInfo.IsCNA)

			if err != nil {
				log.Printf("[ClientRegister] ERROR: Fingerprint validation error for DeviceID=%d: %v", dev.ID(), err)
				dev = nil // Reset dev so STEP_3_CREATE_NEW creates a new device
				goto STEP_3_CREATE_NEW
			}

			if !isValid {
				log.Printf("[ClientRegister] WARN: Fingerprint validation FAILED for MAC-matched DeviceID=%d - possible MAC collision!", dev.ID())
				log.Printf("[ClientRegister] DEBUG: Rejecting MAC match, creating new device")
				dev = nil // Reset dev so STEP_3_CREATE_NEW creates a new device
				goto STEP_3_CREATE_NEW
			}

			log.Printf("[ClientRegister] DEBUG: Fingerprint validation PASSED for DeviceID=%d", dev.ID())

			// Valid fingerprint - update or add it
			if matchedFP != nil {
				err := reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, matchedFP.ID)
				if err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to update fingerprint last_seen for FingerprintID=%d: %v", matchedFP.ID, err)
				}
			} else {
				if err := reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash); err != nil {
					log.Printf("[ClientRegister] ERROR: Failed to add fingerprint for DeviceID=%d: %v", dev.ID(), err)
				}
			}
		} else {
			// No fingerprint data - check if device has stored fingerprints
			storedFingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, dev.ID())
			if err != nil {
				log.Printf("[ClientRegister] ERROR: Failed to check stored fingerprints for DeviceID=%d: %v", dev.ID(), err)
			} else if len(storedFingerprints) > 0 {
				// Device has fingerprints but current request doesn't - SUSPICIOUS for MAC match too!
				log.Printf("[ClientRegister] WARN: Device %d (MAC-matched) has %d stored fingerprint(s) but current request has no fingerprint data - creating new device (possible MAC spoof or JavaScript disabled)", dev.ID(), len(storedFingerprints))
				dev = nil // Reset dev so STEP_3_CREATE_NEW creates a new device
				goto STEP_3_CREATE_NEW
			} else {
				log.Printf("[ClientRegister] DEBUG: Device %d has no stored fingerprints and current request has none - accepting (backward compatibility)", dev.ID())
			}
		}

		// Update network details if changed
		if dev.IpAddr() != params.IpAddr || dev.Hostname() != params.Hostname {
			log.Printf("[ClientRegister] DEBUG: Network details changed, updating device")
			err := reg.UpdateDevice(ctx, clnt, params.MacAddr, params.IpAddr, params.Hostname)
			if err != nil {
				log.Printf("[ClientRegister] ERROR: Failed to update device: %v", err)
				return nil, false, err
			}
		}

		log.Printf("[ClientRegister] SUCCESS: Returned device from MAC match - DeviceID=%d", dev.ID())
		return clnt, true, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ClientRegister] ERROR: Database error searching by MAC=%s: %v", params.MacAddr, err)
	}

STEP_3_CREATE_NEW:
	// Step 3: Not found by cookie or MAC - create new device
	if errors.Is(err, sql.ErrNoRows) || dev == nil {
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

		reg.sessionsMgr.emitClientEvent(sdkapi.EventClientCreated, clnt)
		log.Printf("[ClientRegister] DEBUG: Emitted EventClientCreated for DeviceID=%d", dev.ID())

		// Add first fingerprint for new device (full, partial/CNA, or minimal/JS-disabled)
		if hasFingerprintData && fpHash != "" {
			if hasMinimalFingerprintData {
				log.Printf("[ClientRegister] DEBUG: Adding first MINIMAL fingerprint (JS disabled) for new DeviceID=%d", dev.ID())
			} else if browserInfo.IsCNA && hasCNAFingerprintData {
				log.Printf("[ClientRegister] DEBUG: Adding first PARTIAL fingerprint (CNA) for new DeviceID=%d", dev.ID())
			} else {
				log.Printf("[ClientRegister] DEBUG: Adding first FULL fingerprint for new DeviceID=%d", dev.ID())
			}
			if err := reg.addFingerprint(ctx, dev.ID(), params, browserInfo, fpHash); err != nil {
				log.Printf("[ClientRegister] ERROR: Failed to add first fingerprint for DeviceID=%d: %v", dev.ID(), err)
				// Don't fail registration if fingerprint creation fails
			}
		} else if !hasFingerprintData {
			log.Printf("[ClientRegister] DEBUG: Skipping fingerprint for new DeviceID=%d - no fingerprint data provided", dev.ID())
		}

		return clnt, true, nil
	}

	log.Printf("[ClientRegister] ERROR: Unexpected error: %v", err)
	return nil, false, err
}

// validateDeviceFingerprint checks if current fingerprint matches any stored fingerprints
// Returns (isValid, matchedFingerprint, error)
func (reg *ClientRegister) validateDeviceFingerprint(ctx context.Context, deviceID int64, currentHash string, currentScreen string, currentOS string, currentLang string, currentTZ string, currentIsCNA bool) (bool, *queries.DeviceFingerprint, error) {
	log.Printf("[Fingerprint] Validating fingerprint for DeviceID=%d, Hash=%s, OS=%s, Screen=%s, Lang=%s, TZ=%s, IsCNA=%v",
		deviceID, currentHash[:16]+"...", currentOS, currentScreen, currentLang, currentTZ, currentIsCNA)

	// Get all fingerprints for device (within 6 months)
	fingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, deviceID)
	if err != nil {
		log.Printf("[Fingerprint] ERROR: Failed to fetch fingerprints for DeviceID=%d: %v", deviceID, err)
		return false, nil, err
	}

	if len(fingerprints) == 0 {
		log.Printf("[Fingerprint] No fingerprints stored for DeviceID=%d - first time registration, accepting", deviceID)
		return true, nil, nil
	}

	log.Printf("[Fingerprint] Found %d stored fingerprint(s) for DeviceID=%d, checking matches...", len(fingerprints), deviceID)

	// Check against all stored fingerprints
	for i := range fingerprints {
		fp := &fingerprints[i]
		log.Printf("[Fingerprint] Checking against fingerprint #%d: Hash=%s, OS=%s, Screen=%s, Lang=%s, Browser=%s, IsCNA=%v",
			i+1, fp.FingerprintHash[:16]+"...", fp.OsFamily, fp.ScreenResolution, fp.Language, fp.BrowserName, fp.IsCna)

		result := fingerprint.ValidateFingerprint(
			fingerprint.StoredFingerprint{
				FingerprintHash:  fp.FingerprintHash,
				OSFamily:         fp.OsFamily,
				ScreenResolution: fp.ScreenResolution,
				Language:         fp.Language,
				Timezone:         fp.Timezone, // Now stored in database
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
			log.Printf("[Fingerprint] ✓ EXACT MATCH found! FingerprintID=%d, DeviceID=%d", fp.ID, deviceID)
			return true, fp, nil
		}

		if result == fingerprint.SmartMatch {
			// Log different match scenarios
			if fp.IsCna && currentIsCNA {
				log.Printf("[Fingerprint] ✓ SMART MATCH found! CNA-to-CNA match (same OS). FingerprintID=%d, DeviceID=%d", fp.ID, deviceID)
			} else if fp.IsCna && !currentIsCNA {
				log.Printf("[Fingerprint] ✓ SMART MATCH found! CNA fingerprint accepting browser (same OS). FingerprintID=%d, DeviceID=%d", fp.ID, deviceID)
			} else if !fp.IsCna && currentIsCNA {
				log.Printf("[Fingerprint] ✓ SMART MATCH found! Browser fingerprint accepting CNA (same OS). FingerprintID=%d, DeviceID=%d", fp.ID, deviceID)
			} else {
				log.Printf("[Fingerprint] ✓ SMART MATCH found! Same OS+Screen+Lang+TZ but different hash (browser switch detected). FingerprintID=%d, DeviceID=%d", fp.ID, deviceID)
			}
			return true, nil, nil
		}

		log.Printf("[Fingerprint] ✗ No match for fingerprint (%d/%d)", i+1, len(fingerprints))
	}

	// No match found - different device
	log.Printf("[Fingerprint] ✗ VALIDATION FAILED! No matching fingerprint found for DeviceID=%d", deviceID)
	log.Printf("[Fingerprint] Current: OS=%s, Screen=%s, Lang=%s, TZ=%s | Stored fingerprints had different combinations", currentOS, currentScreen, currentLang, currentTZ)
	return false, nil, nil
}

// addFingerprint creates a new fingerprint record for device
func (reg *ClientRegister) addFingerprint(ctx context.Context, deviceID int64, params ClientRegisterParams, browserInfo browserdetect.BrowserInfo, fpHash string) error {
	// Don't store empty fingerprint hashes
	if fpHash == "" {
		log.Printf("[Fingerprint] WARN: Attempted to add empty fingerprint hash for DeviceID=%d - skipping", deviceID)
		return fmt.Errorf("fingerprint hash is empty")
	}

	log.Printf("[Fingerprint] Adding fingerprint for DeviceID=%d, Hash=%s, Browser=%s, OS=%s, Screen=%s, IsCNA=%v",
		deviceID, fpHash[:16]+"...", browserInfo.BrowserName, browserInfo.OSFamily, params.ScreenRes, browserInfo.IsCNA)

	// Check if exact fingerprint already exists
	existing, err := reg.mdls.DeviceFingerprint().CheckExactMatch(ctx, deviceID, fpHash)
	if err != nil {
		log.Printf("[Fingerprint] ERROR: Failed to check for existing fingerprint: %v", err)
	}

	if existing != nil {
		log.Printf("[Fingerprint] Fingerprint already exists (FingerprintID=%d), updating last_seen timestamp", existing.ID)
		err := reg.mdls.DeviceFingerprint().UpdateLastSeen(ctx, existing.ID)
		if err != nil {
			log.Printf("[Fingerprint] ERROR: Failed to update last_seen for FingerprintID=%d: %v", existing.ID, err)
			return err
		}
		log.Printf("[Fingerprint] ✓ Successfully updated last_seen for FingerprintID=%d", existing.ID)
		return nil
	}

	// Check fingerprint limit (max 10 per device to prevent flooding)
	fingerprints, err := reg.mdls.DeviceFingerprint().FindByDeviceID(ctx, deviceID)
	if err != nil {
		log.Printf("[Fingerprint] ERROR: Failed to count fingerprints for DeviceID=%d: %v", deviceID, err)
		// Continue anyway - don't block on this check
	} else if len(fingerprints) >= 10 {
		log.Printf("[Fingerprint] WARN: Device %d already has %d fingerprints (limit reached) - rejecting new fingerprint to prevent flooding", deviceID, len(fingerprints))
		return fmt.Errorf("fingerprint limit reached (max 10 per device)")
	}

	// Create new fingerprint record
	log.Printf("[Fingerprint] Creating new fingerprint record for DeviceID=%d (current count: %d)", deviceID, len(fingerprints))
	fpID, err := reg.mdls.DeviceFingerprint().Create(ctx, queries.CreateDeviceFingerprintParams{
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

	if err != nil {
		log.Printf("[Fingerprint] ERROR: Failed to create fingerprint for DeviceID=%d: %v", deviceID, err)
		return err
	}

	log.Printf("[Fingerprint] ✓ Successfully created new fingerprint! FingerprintID=%d, DeviceID=%d, Browser=%s, OS=%s, IsCNA=%v",
		fpID, deviceID, browserInfo.BrowserName, browserInfo.OSFamily, browserInfo.IsCNA)
	return nil
}
