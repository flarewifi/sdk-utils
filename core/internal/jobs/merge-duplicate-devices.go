package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"core/db/queries"
	"core/internal/api"
	"core/internal/modules/fingerprint"
	jobque "core/utils/job-que"
)

// mergeQueue serializes concurrent merge runs (queue size 1 — second run
// waits for the first to finish rather than running in parallel).
var mergeQueue = jobque.NewJobQueue[struct{}](1)

// MergeCandidate represents a device being evaluated for merging
type MergeCandidate struct {
	DeviceID     int64
	Fingerprints []queries.DeviceFingerprint
	LastActivity time.Time
	CurrentMAC   string // current MAC address (is_current=TRUE in device_macs)
	CurrentIP    string // current IP address (devices.ip_address)
	Hostname     string // hostname (devices.hostname), may be empty
}

// MergeDecision contains the result of evaluating two devices for merging
type MergeDecision struct {
	ShouldMerge bool
	TargetID    int64  // Device to keep
	SourceID    int64  // Device to merge into target
	Reason      string // Explanation for the decision
}

// MatchKind describes how two fingerprint sets matched.
type MatchKind int

const (
	MatchNone    MatchKind = iota // fingerprints do not match
	MatchBrowser                  // ExactMatch or browser SmartMatch (OS+screen+lang)
	MatchCNA                      // CNA-CNA SmartMatch (OS-family only — weaker signal)
)

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

			if len(deviceIDs) < 2 {
				continue
			}

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
// Rules:
// - If both devices have fingerprints: merge only if fingerprints match
// - If one device has no fingerprints: merge it into the device with fingerprints
// - If neither device has fingerprints: merge based on activity (keep most recent)
// The device with most recent session activity is kept when both are equal candidates.
//
// For CNA fingerprint matches (OS-family-only SmartMatch), the merge is only approved
// when both devices currently share the same MAC address and IP address. If both devices
// have a non-empty hostname, those must also match. This prevents merging two different
// physical devices that happen to share an OS family.
//
// The externalMerged set contains source device IDs already consumed by a prior pass;
// pairs involving those IDs are skipped.
//
// Returns (mergeCount, newlyMergedSources) so the caller can update its tracking set.
func mergeMatchingDevicesTracked(ctx context.Context, g *api.CoreGlobals, deviceIDs []int64, externalMerged map[int64]bool) (int, map[int64]bool) {
	mergeCount := 0
	newlyMergedSources := make(map[int64]bool)

	// Load fingerprints for all devices.
	// If loading fails for a device, mark it with a sentinel so we can skip pairs
	// involving it rather than treating it as fingerprint-less (which would cause
	// wrong merge decisions).
	deviceFPs := make(map[int64][]queries.DeviceFingerprint)
	fpLoadFailed := make(map[int64]bool)
	for _, devID := range deviceIDs {
		fps, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, devID)
		if err != nil {
			log.Printf("[DeviceMerge] WARN: Failed to load fingerprints for device %d, skipping pairs involving it: %v", devID, err)
			fpLoadFailed[devID] = true
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
			decision := shouldMergeDevices(
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

// shouldMergeDevices determines if two devices should be merged based on fingerprints.
func shouldMergeDevices(deviceA, deviceB MergeCandidate) MergeDecision {
	hasA := len(deviceA.Fingerprints) > 0
	hasB := len(deviceB.Fingerprints) > 0

	// Case 1: Both have fingerprints - only merge if they match
	if hasA && hasB {
		kind := fingerprintsMatchKind(deviceA.Fingerprints, deviceB.Fingerprints)
		if kind == MatchNone {
			return MergeDecision{ShouldMerge: false}
		}

		// CNA-CNA SmartMatch is OS-family-only — require same current MAC and IP
		// (and hostname when both non-empty) to guard against false positives.
		if kind == MatchCNA && !cnaMACIPMatch(deviceA, deviceB) {
			log.Printf("[DeviceMerge] Skipping CNA pair (%d, %d): MAC/IP/hostname mismatch (mac: %q vs %q, ip: %q vs %q, hostname: %q vs %q)",
				deviceA.DeviceID, deviceB.DeviceID,
				deviceA.CurrentMAC, deviceB.CurrentMAC,
				deviceA.CurrentIP, deviceB.CurrentIP,
				deviceA.Hostname, deviceB.Hostname,
			)
			return MergeDecision{ShouldMerge: false}
		}

		reason := "fingerprint match"
		if kind == MatchCNA {
			reason = "CNA fingerprint match with same MAC, IP, and hostname"
		}

		// Keep device with most recent activity
		if deviceB.LastActivity.After(deviceA.LastActivity) {
			return MergeDecision{
				ShouldMerge: true,
				TargetID:    deviceB.DeviceID,
				SourceID:    deviceA.DeviceID,
				Reason:      reason,
			}
		}
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceA.DeviceID,
			SourceID:    deviceB.DeviceID,
			Reason:      reason,
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

	// Case 4: Neither has fingerprints - merge based on activity (keep most recent).
	// When both have zero/equal activity, keep deviceA (lower index, deterministic).
	if deviceB.LastActivity.After(deviceA.LastActivity) {
		return MergeDecision{
			ShouldMerge: true,
			TargetID:    deviceB.DeviceID,
			SourceID:    deviceA.DeviceID,
			Reason:      "no fingerprints on either, keeping device with more recent activity",
		}
	}
	reason := "no fingerprints on either, keeping device with more recent activity"
	if deviceA.LastActivity.Equal(deviceB.LastActivity) {
		reason = "no fingerprints or session activity on either, keeping device by insertion order"
	}
	return MergeDecision{
		ShouldMerge: true,
		TargetID:    deviceA.DeviceID,
		SourceID:    deviceB.DeviceID,
		Reason:      reason,
	}
}

// fingerprintsMatchKind checks if two devices have matching fingerprints and returns
// the kind of match found.
//
// CNA guard: when either fingerprint is a CNA fingerprint, ValidateFingerprint
// may return SmartMatch based solely on OS family — which is too permissive for
// a destructive merge operation. In that case we require either an ExactMatch
// (hash collision unlikely but correct) or that BOTH fingerprints are CNA with
// the same OS family (same device class, acceptable risk — caller must additionally
// verify MAC+IP identity via cnaMACIPMatch).
func fingerprintsMatchKind(fpsA, fpsB []queries.DeviceFingerprint) MatchKind {
	best := MatchNone
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

			if result == fingerprint.ExactMatch {
				return MatchBrowser // ExactMatch is always definitive
			}

			if result == fingerprint.SmartMatch {
				if a.IsCna && b.IsCna {
					// CNA-CNA SmartMatch: OS-family-only — weaker signal.
					// Caller must verify MAC+IP before acting on this.
					if best < MatchCNA {
						best = MatchCNA
					}
				} else if !a.IsCna && !b.IsCna {
					// Regular browser SmartMatch (OS + screen + language) is acceptable.
					return MatchBrowser
				} else {
					// One CNA, one browser: SmartMatch is OS-only — too weak for a merge.
					log.Printf("[DeviceMerge] Skipping CNA↔browser SmartMatch (too permissive for merge): isCNA=(%v,%v) os=(%s,%s)", a.IsCna, b.IsCna, a.OsFamily, b.OsFamily)
				}
			}
		}
	}
	return best
}

// cnaMACIPMatch returns true when two CNA-matched candidates share the same
// current MAC address and IP address. If both devices have a non-empty hostname,
// those must also match.
//
// An empty MAC or IP on either side causes an automatic false — a device with no
// current MAC/IP record cannot be safely merged on CNA evidence alone.
func cnaMACIPMatch(a, b MergeCandidate) bool {
	if a.CurrentMAC == "" || b.CurrentMAC == "" {
		return false
	}
	if a.CurrentIP == "" || b.CurrentIP == "" {
		return false
	}
	if a.CurrentMAC != b.CurrentMAC {
		return false
	}
	if a.CurrentIP != b.CurrentIP {
		return false
	}
	// If both devices have a non-empty hostname, they must match.
	if a.Hostname != "" && b.Hostname != "" && a.Hostname != b.Hostname {
		return false
	}
	return true
}

// parseSQLiteTime attempts to parse a timestamp string returned by SQLite,
// trying multiple formats to handle fractional seconds and RFC3339 variants.
func parseSQLiteTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05.999999999", // fractional seconds (space separator)
		"2006-01-02 15:04:05",           // standard SQLite format
		time.RFC3339Nano,                // "2006-01-02T15:04:05.999999999Z07:00"
		time.RFC3339,                    // "2006-01-02T15:04:05Z07:00"
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
