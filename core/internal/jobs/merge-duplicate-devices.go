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

// Type aliases for convenience — merge decision types live in the fingerprint package
// so they can be reused by the registration flow (UpdateDevice inline merge).
type MergeCandidate = fingerprint.MergeCandidate
type MergeDecision = fingerprint.MergeDecision

// StartDeviceMergeScheduler starts a background goroutine that merges
// duplicate devices. Devices are identified by shared MAC addresses or identical
// fingerprint hashes, and merged only if their fingerprints match (same physical device).
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

// performDeviceMerge finds devices with shared MAC history or identical fingerprint hashes
// and merges those with matching fingerprints. This handles cases where MAC randomization
// created multiple device records for the same physical device.
//
// Two passes are performed:
//  1. MAC pass: devices that share a MAC address in their history (existing behaviour).
//  2. Fingerprint-hash pass: devices that share an identical non-CNA fingerprint hash but
//     may never have shared a MAC (e.g. complete MAC randomization on every connection).
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

	// mergedSources tracks source device IDs that were successfully merged in either pass,
	// so the fingerprint-hash pass does not attempt to reprocess already-merged devices.
	mergedSources := make(map[int64]bool)

	// Calculate lookback window: 30 days ago in UTC
	sinceUtcTime := time.Now().UTC().AddDate(0, 0, -30)
	sinceUtc := sql.NullTime{Time: sinceUtcTime, Valid: true}

	// -------------------------------------------------------------------------
	// Pass 1: Shared MAC addresses
	// -------------------------------------------------------------------------

	sharedMacs, err := g.Database.Queries.FindSharedMacAddresses(ctx, sinceUtc)
	if err != nil {
		log.Printf("[DeviceMerge] ERROR: Failed to find shared MACs: %v", err)
		return
	}

	if len(sharedMacs) == 0 {
		log.Println("[DeviceMerge] Pass 1: No shared MAC addresses found")
	} else {
		log.Printf("[DeviceMerge] Pass 1: Found %d shared MAC address(es) to process", len(sharedMacs))

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

			log.Printf("[DeviceMerge] DEBUG: Calling mergeMatchingDevicesTracked for devices: %v", deviceIDs)
			merged, sources := mergeMatchingDevicesTracked(ctx, g, deviceIDs, mergedSources)
			mergeCount += merged
			for id := range sources {
				mergedSources[id] = true
			}
		}
	}

	// -------------------------------------------------------------------------
	// Pass 2: Shared fingerprint hashes (non-CNA only)
	// -------------------------------------------------------------------------

	sharedHashes, err := g.Models.DeviceFingerprint().FindSharedFingerprintHashes(ctx, sinceUtcTime)
	if err != nil {
		log.Printf("[DeviceMerge] ERROR: Failed to find shared fingerprint hashes: %v", err)
		// Don't return — MAC pass results are still valid; log and skip pass 2.
	} else if len(sharedHashes) == 0 {
		log.Println("[DeviceMerge] Pass 2: No shared fingerprint hashes found")
	} else {
		log.Printf("[DeviceMerge] Pass 2: Found %d shared fingerprint hash(es) to process", len(sharedHashes))

		for _, hash := range sharedHashes {
			deviceIDs, err := g.Models.DeviceFingerprint().FindDeviceIDsByFingerprintHash(ctx, hash, sinceUtcTime)
			if err != nil {
				log.Printf("[DeviceMerge] WARN: Failed to get devices for fingerprint hash %s: %v", hash, err)
				continue
			}

			// Filter out devices already merged in pass 1 (or earlier in pass 2)
			filtered := make([]int64, 0, len(deviceIDs))
			for _, id := range deviceIDs {
				if !mergedSources[id] {
					filtered = append(filtered, id)
				}
			}

			if len(filtered) < 2 {
				continue
			}

			merged, sources := mergeMatchingDevicesTracked(ctx, g, filtered, mergedSources)
			mergeCount += merged
			for id := range sources {
				mergedSources[id] = true
			}
		}
	}

	duration := time.Since(startTime)
	log.Printf("[DeviceMerge] Completed in %v, merged %d device pair(s)",
		duration.Round(time.Millisecond), mergeCount)
}

// mergeMatchingDevicesTracked compares fingerprints for a group of devices and merges matches.
// Uses fingerprint.ShouldMergeDevices for the merge decision (shared with UpdateDevice inline merge).
//
// The externalMerged set contains source device IDs already consumed by a prior pass;
// pairs involving those IDs are skipped.
//
// Returns (mergeCount, newlyMergedSources) so the caller can update its tracking set.
func mergeMatchingDevicesTracked(ctx context.Context, g *api.CoreGlobals, deviceIDs []int64, externalMerged map[int64]bool) (int, map[int64]bool) {
	mergeCount := 0
	newlyMergedSources := make(map[int64]bool)

	// Load fingerprints for all devices and convert to StoredFingerprint for merge comparison.
	// If loading fails for a device, mark it with a sentinel so we can skip pairs
	// involving it rather than treating it as fingerprint-less (which would cause
	// wrong merge decisions).
	deviceFPs := make(map[int64][]fingerprint.StoredFingerprint)
	fpLoadFailed := make(map[int64]bool)
	for _, devID := range deviceIDs {
		fps, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, devID)
		if err != nil {
			log.Printf("[DeviceMerge] WARN: Failed to load fingerprints for device %d, skipping pairs involving it: %v", devID, err)
			fpLoadFailed[devID] = true
			continue
		}
		records := make([]fingerprint.FingerprintRecord, len(fps))
		for i, fp := range fps {
			records[i] = fingerprint.FingerprintRecord{
				FingerprintHash:  fp.FingerprintHash,
				OsFamily:         fp.OsFamily,
				ScreenResolution: fp.ScreenResolution,
				Language:         fp.Language,
				Timezone:         fp.Timezone,
				IsCna:            fp.IsCna,
			}
		}
		deviceFPs[devID] = fingerprint.ToStoredFingerprints(records)
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
				// SQLite may return time as string in several formats
				deviceActivity[devID] = parseSQLiteTime(s)
			} else {
				deviceActivity[devID] = time.Time{}
			}
		} else {
			deviceActivity[devID] = time.Time{} // Zero time if no sessions
		}
	}

	// Load current network identity (MAC, IP, hostname) for each device.
	// Used to enforce MAC+IP matching when the only fingerprint signal is CNA.
	// A load failure is non-fatal: the candidate will have empty fields, which
	// causes cnaMACIPMatch to return false (safe fallback — no merge).
	deviceNetID := make(map[int64]struct {
		mac      string
		ip       string
		hostname string
	})
	for _, devID := range deviceIDs {
		dev, err := g.Models.Device().Find(ctx, devID)
		if err != nil {
			log.Printf("[DeviceMerge] WARN: Failed to load network identity for device %d: %v", devID, err)
			// Leave zero-value entry — cnaMACIPMatch will reject this pair safely.
			deviceNetID[devID] = struct{ mac, ip, hostname string }{}
			continue
		}
		deviceNetID[devID] = struct{ mac, ip, hostname string }{
			mac:      dev.MacAddr(),
			ip:       dev.IpAddr(),
			hostname: dev.Hostname(),
		}
	}

	// Compare pairs
	for i := 0; i < len(deviceIDs); i++ {
		devA := deviceIDs[i]
		if newlyMergedSources[devA] || externalMerged[devA] {
			continue
		}

		for j := i + 1; j < len(deviceIDs); j++ {
			devB := deviceIDs[j]
			if newlyMergedSources[devB] || externalMerged[devB] {
				continue
			}

			// Skip pairs where fingerprint loading failed for either device —
			// we cannot safely determine whether they match.
			if fpLoadFailed[devA] || fpLoadFailed[devB] {
				log.Printf("[DeviceMerge] Skipping pair (%d, %d): fingerprint load failed for one or both devices", devA, devB)
				continue
			}

			netA := deviceNetID[devA]
			netB := deviceNetID[devB]

			// Determine if we should merge and which device to keep
			decision := fingerprint.ShouldMergeDevices(
				MergeCandidate{
					DeviceID:     devA,
					Fingerprints: deviceFPs[devA],
					LastActivity: deviceActivity[devA],
					CurrentMAC:   netA.mac,
					CurrentIP:    netA.ip,
					Hostname:     netA.hostname,
				},
				MergeCandidate{
					DeviceID:     devB,
					Fingerprints: deviceFPs[devB],
					LastActivity: deviceActivity[devB],
					CurrentMAC:   netB.mac,
					CurrentIP:    netB.ip,
					Hostname:     netB.hostname,
				},
			)

			if !decision.ShouldMerge {
				log.Printf("[DeviceMerge] DEBUG: Not merging devices %d and %d (reason: fingerprints don't match or other criteria)", devA, devB)
				continue
			}

			log.Printf("[DeviceMerge] Merging device %d into %d (%s)", decision.SourceID, decision.TargetID, decision.Reason)

			if err := g.ClientMgr.MergeClientDevices(ctx, decision.TargetID, decision.SourceID); err != nil {
				log.Printf("[DeviceMerge] ERROR: Failed to merge device %d into %d: %v", decision.SourceID, decision.TargetID, err)
				continue
			}

			newlyMergedSources[decision.SourceID] = true
			mergeCount++
		}
	}

	return mergeCount, newlyMergedSources
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
