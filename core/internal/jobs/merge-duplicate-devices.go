package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"core/internal/api"
	"core/internal/modules/fingerprint"
	jobque "core/utils/job-que"
)

// mergeQueue serializes concurrent merge runs (queue size 1 — second run
// waits for the first to finish rather than running in parallel).
var mergeQueue = jobque.NewJobQueue[struct{}](1)

// StartDeviceMergeScheduler starts a background goroutine that merges
// duplicate devices. Devices are identified by shared MAC addresses
// and merged only if their fingerprints match (same physical device).
//
// A merge run is performed immediately on boot, then scheduled:
//   - Dev mode: every 5 seconds.
//   - Production mode: daily at 3:30 AM.
func StartDeviceMergeScheduler(g *api.CoreGlobals) {
	go func() {
		// Run once on boot to merge any duplicates that accumulated while the
		// server was offline or before this feature was deployed.
		log.Println("[DeviceMerge] Running initial merge on boot")
		performDeviceMerge(g)

		// Dev mode: run at fixed interval
		if DeviceMergeInterval > 0 {
			log.Printf("[DeviceMerge] DEV MODE: Running every %v", DeviceMergeInterval)
			for {
				time.Sleep(DeviceMergeInterval)
				performDeviceMerge(g)
			}
			return // unreachable, defensive guard against future refactors
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

// performDeviceMerge finds devices with shared MAC history and merges those with matching
// fingerprints. This handles cases where MAC randomization created multiple device records
// for the same physical device.
//
// Concurrent calls are serialized via mergeQueue (queue size 1): if a run is already in
// progress the new call waits rather than running in parallel, preventing races on the
// same device pairs.
func performDeviceMerge(g *api.CoreGlobals) {
	_, _ = mergeQueue.Exec("DeviceMerge", func() (struct{}, error) {
		runMerge(g)
		return struct{}{}, nil
	})
}

func runMerge(g *api.CoreGlobals) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("[DeviceMerge] Starting device merge scan")
	startTime := time.Now()
	mergeCount := 0

	// Calculate lookback window: 30 days ago in UTC
	sinceUtc := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -30), Valid: true}

	sharedMacs, err := g.Database.Queries.FindSharedMacAddresses(ctx, sinceUtc)
	if err != nil {
		log.Printf("[DeviceMerge] ERROR: Failed to find shared MACs: %v", err)
		return
	}

	if len(sharedMacs) == 0 {
		log.Println("[DeviceMerge] No shared MAC addresses found")
	} else {
		log.Printf("[DeviceMerge] Found %d shared MAC address(es) to process", len(sharedMacs))

		for _, mac := range sharedMacs {
			deviceIDs, err := g.Database.Queries.FindDeviceIDsByMacAddress(ctx, mac)
			if err != nil {
				log.Printf("[DeviceMerge] WARN: Failed to get devices for MAC %s: %v", mac, err)
				continue
			}

			log.Printf("[DeviceMerge] DEBUG: Found %d device(s) sharing MAC %s: %v", len(deviceIDs), mac, deviceIDs)

			if len(deviceIDs) < 2 {
				log.Printf("[DeviceMerge] DEBUG: Skipping MAC %s - only %d device(s)", mac, len(deviceIDs))
				continue
			}

			log.Printf("[DeviceMerge] DEBUG: Calling mergeMatchingDevices for devices: %v", deviceIDs)
			mergeCount += mergeMatchingDevices(ctx, g, deviceIDs)
		}
	}

	duration := time.Since(startTime)
	log.Printf("[DeviceMerge] Completed in %v, merged %d device pair(s)",
		duration.Round(time.Millisecond), mergeCount)
}

// mergeMatchingDevices compares full browser fingerprints for a group of devices
// sharing a MAC address and merges those that match.
// Only considers non-CNA (browser) fingerprints — CNA fingerprints (OS-only) are
// too weak to reliably identify a device for a destructive merge.
func mergeMatchingDevices(ctx context.Context, g *api.CoreGlobals, deviceIDs []int64) int {
	mergeCount := 0
	mergedSources := make(map[int64]bool)

	// Load browser fingerprints for all devices (skip CNA-only fingerprints).
	type deviceFPData struct {
		fps    []fingerprint.StoredFingerprint
		failed bool
	}
	deviceFPs := make(map[int64]deviceFPData)
	for _, devID := range deviceIDs {
		fps, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, devID)
		if err != nil {
			log.Printf("[DeviceMerge] WARN: Failed to load fingerprints for device %d: %v", devID, err)
			deviceFPs[devID] = deviceFPData{failed: true}
			continue
		}
		// Filter to browser-only fingerprints
		var browserFPs []fingerprint.StoredFingerprint
		for _, fp := range fps {
			if !fp.IsCna {
				browserFPs = append(browserFPs, fingerprint.StoredFingerprint{
					FingerprintHash:  fp.FingerprintHash,
					OSFamily:         fp.OsFamily,
					ScreenResolution: fp.ScreenResolution,
					Language:         fp.Language,
					Timezone:         fp.Timezone,
					IsCna:            false,
				})
			}
		}
		deviceFPs[devID] = deviceFPData{fps: browserFPs}
	}

	// Get most recent activity for each device (for determining which to keep)
	deviceActivity := make(map[int64]time.Time)
	for _, devID := range deviceIDs {
		activity, err := g.Database.Queries.GetMostRecentSessionTimeForDevice(ctx, devID)
		if err == nil && activity != nil {
			if t, ok := activity.(time.Time); ok {
				deviceActivity[devID] = t
			} else if s, ok := activity.(string); ok {
				deviceActivity[devID] = parseSQLiteTime(s)
			} else {
				deviceActivity[devID] = time.Time{}
			}
		} else {
			deviceActivity[devID] = time.Time{}
		}
	}

	// Compare pairs — only merge when both have browser fingerprints that match
	for i := 0; i < len(deviceIDs); i++ {
		devA := deviceIDs[i]
		if mergedSources[devA] {
			continue
		}
		dataA := deviceFPs[devA]
		if dataA.failed || len(dataA.fps) == 0 {
			continue // No browser fingerprints — skip
		}

		for j := i + 1; j < len(deviceIDs); j++ {
			devB := deviceIDs[j]
			if mergedSources[devB] {
				continue
			}
			dataB := deviceFPs[devB]
			if dataB.failed || len(dataB.fps) == 0 {
				continue // No browser fingerprints — skip
			}

			// Check if any browser fingerprint pair matches
			if !browserFingerprintsMatch(dataA.fps, dataB.fps) {
				continue
			}

			// Determine target (keep device with most recent activity)
			targetID, sourceID := devA, devB
			if deviceActivity[devB].After(deviceActivity[devA]) {
				targetID, sourceID = devB, devA
			}

			log.Printf("[DeviceMerge] Merging device %d into %d (browser fingerprint match)", sourceID, targetID)

			if err := g.ClientMgr.MergeClientDevices(ctx, targetID, sourceID); err != nil {
				log.Printf("[DeviceMerge] ERROR: Failed to merge device %d into %d: %v", sourceID, targetID, err)
				continue
			}

			mergedSources[sourceID] = true
			mergeCount++
		}
	}

	return mergeCount
}

// browserFingerprintsMatch checks if two sets of browser fingerprints have a match
// (ExactMatch or SmartMatch via ValidateFingerprint).
func browserFingerprintsMatch(fpsA, fpsB []fingerprint.StoredFingerprint) bool {
	for _, a := range fpsA {
		for _, b := range fpsB {
			result := fingerprint.ValidateFingerprint(a, b.FingerprintHash, b.OSFamily, b.ScreenResolution, b.Language, b.Timezone, b.IsCna)
			if result == fingerprint.ExactMatch || result == fingerprint.SmartMatch {
				return true
			}
		}
	}
	return false
}

// parseSQLiteTime attempts to parse a timestamp string returned by SQLite,
// trying multiple formats to handle fractional seconds and RFC3339 variants.
func parseSQLiteTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00", // fractional seconds with timezone (space separator)
		"2006-01-02 15:04:05.999999999",       // fractional seconds (space separator)
		"2006-01-02 15:04:05",                 // standard SQLite format
		time.RFC3339Nano,                      // "2006-01-02T15:04:05.999999999Z07:00"
		time.RFC3339,                          // "2006-01-02T15:04:05Z07:00"
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	log.Printf("[DeviceMerge] WARN: Could not parse SQLite timestamp %q with any known format, treating as zero time", s)
	return time.Time{}
}

// RunDeviceMergeNow executes merge immediately (useful for manual triggers or testing)
func RunDeviceMergeNow(g *api.CoreGlobals) {
	log.Println("[DeviceMerge] Manual merge triggered")
	performDeviceMerge(g)
}
