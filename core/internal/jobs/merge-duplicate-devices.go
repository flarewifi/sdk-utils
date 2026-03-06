package jobs

import (
	"context"
	"log"
	"time"

	"core/db/queries"
	"core/internal/api"
	"core/internal/modules/fingerprint"
)

// MergeCandidate represents a device being evaluated for merging
type MergeCandidate struct {
	DeviceID     int64
	Fingerprints []queries.DeviceFingerprint
	LastActivity time.Time
}

// MergeDecision contains the result of evaluating two devices for merging
type MergeDecision struct {
	ShouldMerge bool
	TargetID    int64  // Device to keep
	SourceID    int64  // Device to merge into target
	Reason      string // Explanation for the decision
}

// StartDeviceMergeScheduler starts a background goroutine that merges
// duplicate devices. Devices are identified by shared MAC addresses in their
// history, and merged only if their fingerprints match (same physical device).
// In dev mode: runs every 5 seconds. In prod: runs daily at 3:30 AM.
func StartDeviceMergeScheduler(g *api.CoreGlobals) {
	go func() {
		// Dev mode: run at fixed interval
		if DeviceMergeInterval > 0 {
			log.Printf("[DeviceMerge] DEV MODE: Running every %v", DeviceMergeInterval)
			for {
				time.Sleep(DeviceMergeInterval)
				performDeviceMerge(g)
			}
		}

		// Production mode: run at specific time daily
		log.Printf("[DeviceMerge] Scheduler started - will run daily at %d:%02d AM",
			DeviceMergeHour, DeviceMergeMinute)

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(),
				DeviceMergeHour, DeviceMergeMinute, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			waitDuration := next.Sub(now)
			log.Printf("[DeviceMerge] Next merge scheduled in %v (at %s)",
				waitDuration.Round(time.Second), next.Format("2006-01-02 15:04:05"))

			time.Sleep(waitDuration)
			performDeviceMerge(g)
		}
	}()
}

// performDeviceMerge finds devices with shared MAC history and merges
// those with matching fingerprints. This handles cases where MAC randomization
// created multiple device records for the same physical device.
func performDeviceMerge(g *api.CoreGlobals) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("[DeviceMerge] Starting device merge scan")
	startTime := time.Now()
	mergeCount := 0

	// 1. Find MAC addresses shared by multiple devices (within 30 days)
	sharedMacs, err := g.Database.Queries.FindSharedMacAddresses(ctx)
	if err != nil {
		log.Printf("[DeviceMerge] ERROR: Failed to find shared MACs: %v", err)
		return
	}

	if len(sharedMacs) == 0 {
		log.Println("[DeviceMerge] No shared MAC addresses found")
		return
	}

	log.Printf("[DeviceMerge] Found %d shared MAC address(es) to process", len(sharedMacs))

	// 2. Process each shared MAC
	for _, mac := range sharedMacs {
		deviceIDs, err := g.Database.Queries.FindDeviceIDsByMacAddress(ctx, mac)
		if err != nil {
			log.Printf("[DeviceMerge] WARN: Failed to get devices for MAC %s: %v", mac, err)
			continue
		}

		if len(deviceIDs) < 2 {
			continue
		}

		// 3. Compare fingerprints and merge matching devices
		merged := mergeMatchingDevices(ctx, g, deviceIDs)
		mergeCount += merged
	}

	duration := time.Since(startTime)
	log.Printf("[DeviceMerge] Completed in %v, merged %d device pair(s)",
		duration.Round(time.Millisecond), mergeCount)
}

// mergeMatchingDevices compares fingerprints for a group of devices and merges matches.
// Rules:
// - If both devices have fingerprints: merge only if fingerprints match
// - If one device has no fingerprints: merge it into the device with fingerprints
// - If neither device has fingerprints: merge based on activity (keep most recent)
// The device with most recent session activity is kept when both are equal candidates.
// Returns the number of merges performed.
func mergeMatchingDevices(ctx context.Context, g *api.CoreGlobals, deviceIDs []int64) int {
	mergeCount := 0

	// Load fingerprints for all devices
	deviceFPs := make(map[int64][]queries.DeviceFingerprint)
	for _, devID := range deviceIDs {
		fps, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, devID)
		if err != nil {
			continue
		}
		deviceFPs[devID] = fps
	}

	// Get most recent activity for each device (for determining which to keep)
	deviceActivity := make(map[int64]time.Time)
	for _, devID := range deviceIDs {
		activity, err := g.Database.Queries.GetMostRecentSessionTimeForDevice(ctx, devID)
		if err == nil && activity != nil {
			// Parse the activity time - sqlc returns interface{} for MAX() aggregates
			if t, ok := activity.(time.Time); ok {
				deviceActivity[devID] = t
			} else if s, ok := activity.(string); ok {
				// SQLite may return time as string
				if parsed, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
					deviceActivity[devID] = parsed
				} else {
					deviceActivity[devID] = time.Time{}
				}
			} else {
				deviceActivity[devID] = time.Time{}
			}
		} else {
			deviceActivity[devID] = time.Time{} // Zero time if no sessions
		}
	}

	// Track which devices have been merged (to avoid re-processing)
	merged := make(map[int64]bool)

	// Compare pairs
	for i := 0; i < len(deviceIDs); i++ {
		devA := deviceIDs[i]
		if merged[devA] {
			continue
		}

		for j := i + 1; j < len(deviceIDs); j++ {
			devB := deviceIDs[j]
			if merged[devB] {
				continue
			}

			// Determine if we should merge and which device to keep
			decision := shouldMergeDevices(
				MergeCandidate{
					DeviceID:     devA,
					Fingerprints: deviceFPs[devA],
					LastActivity: deviceActivity[devA],
				},
				MergeCandidate{
					DeviceID:     devB,
					Fingerprints: deviceFPs[devB],
					LastActivity: deviceActivity[devB],
				},
			)

			if !decision.ShouldMerge {
				continue
			}

			log.Printf("[DeviceMerge] Merging device %d into %d (%s)", decision.SourceID, decision.TargetID, decision.Reason)

			// Disconnect source device if it has active session
			sourceClnt, err := g.ClientMgr.FindClientById(ctx, decision.SourceID)
			if err != nil {
				log.Printf("[DeviceMerge] WARN: Failed to find source device %d: %v", decision.SourceID, err)
				continue
			}

			if _, hasSession := g.ClientMgr.GetRunningSession(sourceClnt); hasSession {
				if err := g.ClientMgr.Disconnect(ctx, sourceClnt, ""); err != nil {
					log.Printf("[DeviceMerge] WARN: Failed to disconnect session on device %d: %v", decision.SourceID, err)
					// Continue with merge anyway
				}
			}

			// Perform merge
			if err := g.Models.Device().MergeDevices(ctx, decision.TargetID, decision.SourceID); err != nil {
				log.Printf("[DeviceMerge] ERROR: Failed to merge device %d into %d: %v", decision.SourceID, decision.TargetID, err)
				continue
			}

			merged[decision.SourceID] = true
			mergeCount++
			log.Printf("[DeviceMerge] Successfully merged device %d into %d", decision.SourceID, decision.TargetID)
		}
	}

	return mergeCount
}

// shouldMergeDevices determines if two devices should be merged based on fingerprints.
func shouldMergeDevices(deviceA, deviceB MergeCandidate) MergeDecision {
	hasA := len(deviceA.Fingerprints) > 0
	hasB := len(deviceB.Fingerprints) > 0

	// Case 1: Both have fingerprints - only merge if they match
	if hasA && hasB {
		if !fingerprintsMatch(deviceA.Fingerprints, deviceB.Fingerprints) {
			return MergeDecision{ShouldMerge: false}
		}
		// Keep device with most recent activity
		if deviceB.LastActivity.After(deviceA.LastActivity) {
			return MergeDecision{
				ShouldMerge: true,
				TargetID:    deviceB.DeviceID,
				SourceID:    deviceA.DeviceID,
				Reason:      "fingerprint match",
			}
		}
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceA.DeviceID,
			SourceID:    deviceB.DeviceID,
			Reason:      "fingerprint match",
		}
	}

	// Case 2: Only A has fingerprints - merge B into A
	if hasA && !hasB {
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceA.DeviceID,
			SourceID:    deviceB.DeviceID,
			Reason:      "no fingerprint on source, merging into device with fingerprint",
		}
	}

	// Case 3: Only B has fingerprints - merge A into B
	if !hasA && hasB {
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceB.DeviceID,
			SourceID:    deviceA.DeviceID,
			Reason:      "no fingerprint on source, merging into device with fingerprint",
		}
	}

	// Case 4: Neither has fingerprints - merge based on activity (keep most recent)
	if deviceB.LastActivity.After(deviceA.LastActivity) {
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceB.DeviceID,
			SourceID:    deviceA.DeviceID,
			Reason:      "no fingerprints on either, keeping device with recent activity",
		}
	}
	return MergeDecision{
		ShouldMerge: true,
		TargetID:    deviceA.DeviceID,
		SourceID:    deviceB.DeviceID,
		Reason:      "no fingerprints on either, keeping device with recent activity",
	}
}

// fingerprintsMatch checks if two devices have matching fingerprints.
// Returns true if any fingerprint from device A matches any from device B.
func fingerprintsMatch(fpsA, fpsB []queries.DeviceFingerprint) bool {
	for _, a := range fpsA {
		for _, b := range fpsB {
			result := fingerprint.ValidateFingerprint(
				fingerprint.StoredFingerprint{
					FingerprintHash:  a.FingerprintHash,
					OSFamily:         a.OsFamily,
					ScreenResolution: a.ScreenResolution,
					Language:         a.Language,
					Timezone:         a.Timezone,
					IsCna:            a.IsCna,
				},
				b.FingerprintHash,
				b.OsFamily,
				b.ScreenResolution,
				b.Language,
				b.Timezone,
				b.IsCna,
			)

			if result == fingerprint.ExactMatch || result == fingerprint.SmartMatch {
				return true
			}
		}
	}
	return false
}

// RunDeviceMergeNow executes merge immediately (useful for manual triggers or testing)
func RunDeviceMergeNow(g *api.CoreGlobals) {
	log.Println("[DeviceMerge] Manual merge triggered")
	performDeviceMerge(g)
}
