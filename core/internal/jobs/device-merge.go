package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/api"
	"core/internal/modules/fingerprint"
	"core/internal/sessmgr"
)

// deviceMergeLookbackDays bounds the shared-MAC scan to MACs seen in this window.
// Matches the lookback documented on the FindSharedMacAddresses query.
const deviceMergeLookbackDays = 30

// deviceMergeMaxPassesPerMac caps how many merge passes we make over a single MAC
// group in one run. Each successful merge deletes the source device, so a group of
// N devices converges in at most N-1 passes; this is a defensive upper bound.
const deviceMergeMaxPassesPerMac = 20

// StartDeviceMergeScheduler wires up the device-merge reconciliation job.
//
// Modern client devices randomize their MAC per-SSID and periodically rotate it,
// and captive-portal cookies/tokens are frequently lost (private browsing, CNA
// assistants, cache clears). The live registration path (ClientRegister) can only
// re-identify a device by cookie or a previously-seen MAC — a device that shows up
// with a brand-new MAC and no cookie becomes a NEW device row. This job is the
// offline safety net: it periodically scans MACs shared across multiple device rows
// and merges the rows that fingerprints confirm are the same physical device.
func StartDeviceMergeScheduler(database *db.Database, mdls *models.Models, clientMgr *sessmgr.SessionsMgr, coreAPI *api.PluginApi) {
	go func() {
		time.Sleep(DeviceMergeInitialDelay)
		performDeviceMerge(database, mdls, clientMgr, coreAPI)

		ticker := time.NewTicker(DeviceMergeInterval)
		defer ticker.Stop()

		for range ticker.C {
			performDeviceMerge(database, mdls, clientMgr, coreAPI)
		}
	}()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// performDeviceMerge runs one reconciliation pass over all shared MACs.
func performDeviceMerge(database *db.Database, mdls *models.Models, clientMgr *sessmgr.SessionsMgr, coreAPI *api.PluginApi) {
	// A reconciliation pass must never crash the process — it is best-effort cleanup.
	defer func() { _ = recover() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	since := time.Now().UTC().AddDate(0, 0, -deviceMergeLookbackDays)
	sharedMacs, err := database.Queries.FindSharedMacAddresses(ctx, sql.NullTime{Time: since, Valid: true})
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("device merge: failed to list shared MACs: %v", err))
		return
	}
	if len(sharedMacs) == 0 {
		return
	}

	totalMerged := 0
	for _, mac := range sharedMacs {
		if ctx.Err() != nil {
			break // deadline reached — resume next tick
		}
		totalMerged += reconcileMacGroup(ctx, database, mdls, clientMgr, coreAPI, mac)
	}

	if totalMerged > 0 {
		coreAPI.Logger().Info(fmt.Sprintf("device merge: reconciled %d duplicate device(s) across %d shared MAC(s)", totalMerged, len(sharedMacs)))
	}
}

// reconcileMacGroup merges the confirmed-duplicate devices that share a single MAC.
// It re-reads the group after every merge (a merge deletes the source and moves its
// fingerprints/MACs onto the target), so later comparisons see the combined state.
// Returns the number of merges performed for this MAC.
func reconcileMacGroup(ctx context.Context, database *db.Database, mdls *models.Models, clientMgr *sessmgr.SessionsMgr, coreAPI *api.PluginApi, mac string) int {
	merged := 0

	for pass := 0; pass < deviceMergeMaxPassesPerMac; pass++ {
		if ctx.Err() != nil {
			break
		}

		deviceIDs, err := database.Queries.FindDeviceIDsByMacAddress(ctx, mac)
		if err != nil {
			coreAPI.Logger().Error(fmt.Sprintf("device merge: failed to list devices for a shared MAC: %v", err))
			return merged
		}
		if len(deviceIDs) < 2 {
			return merged // nothing left to reconcile for this MAC
		}

		candidates := make([]fingerprint.MergeCandidate, 0, len(deviceIDs))
		for _, id := range deviceIDs {
			if cand, ok := buildMergeCandidate(ctx, mdls, id); ok {
				candidates = append(candidates, cand)
			}
		}
		if len(candidates) < 2 {
			return merged
		}

		// Find the first mergeable pair, act on it, then restart the pass so the
		// group is re-read with the post-merge state.
		didMerge := false
		for i := 0; i < len(candidates) && !didMerge; i++ {
			for j := i + 1; j < len(candidates); j++ {
				decision := fingerprint.ShouldMergeDevices(candidates[i], candidates[j])
				if !decision.ShouldMerge {
					// A non-empty reason means the fingerprints matched but a guard
					// (e.g. concurrent activity) vetoed the merge — worth surfacing.
					if decision.Reason != "" {
						coreAPI.Logger().Info(fmt.Sprintf("device merge: kept devices %d and %d separate (%s)", candidates[i].DeviceID, candidates[j].DeviceID, decision.Reason))
					}
					continue
				}

				// Don't disrupt live users: defer merging any pair where either
				// device currently has a running session to a future idle pass.
				if hasRunningSession(ctx, clientMgr, decision.TargetID) || hasRunningSession(ctx, clientMgr, decision.SourceID) {
					continue
				}

				if err := clientMgr.MergeClientDevices(ctx, decision.TargetID, decision.SourceID); err != nil {
					coreAPI.Logger().Error(fmt.Sprintf("device merge: failed to merge device %d into %d: %v", decision.SourceID, decision.TargetID, err))
					// Skip this MAC to avoid re-selecting the same failing pair forever.
					return merged
				}

				coreAPI.Logger().Info(fmt.Sprintf("device merge: merged device %d into %d (%s)", decision.SourceID, decision.TargetID, decision.Reason))
				merged++
				didMerge = true
				break
			}
		}

		if !didMerge {
			return merged // group is stable — no more mergeable pairs
		}
	}

	return merged
}

// buildMergeCandidate loads a device and its fingerprints into a MergeCandidate.
// Returns ok=false if the device can't be loaded (e.g. merged away concurrently).
func buildMergeCandidate(ctx context.Context, mdls *models.Models, deviceID int64) (fingerprint.MergeCandidate, bool) {
	dev, err := mdls.Device().Find(ctx, deviceID)
	if err != nil || dev == nil {
		return fingerprint.MergeCandidate{}, false
	}

	fps, err := mdls.DeviceFingerprint().FindByDeviceID(ctx, deviceID)
	if err != nil {
		fps = nil
	}

	records := make([]fingerprint.FingerprintRecord, 0, len(fps))
	lastActivity := dev.UpdatedAt() // fallback when no fingerprint timestamps exist
	for _, fp := range fps {
		records = append(records, fingerprint.FingerprintRecord{
			FingerprintHash:  fp.FingerprintHash,
			OsFamily:         fp.OsFamily,
			ScreenResolution: fp.ScreenResolution,
			Language:         fp.Language,
			Timezone:         fp.Timezone,
			IsCna:            fp.IsCna,
		})
		if fp.LastSeenAt.Valid && fp.LastSeenAt.Time.After(lastActivity) {
			lastActivity = fp.LastSeenAt.Time
		}
	}

	currentIP := dev.Ipv4Addr()
	if currentIP == "" {
		currentIP = dev.Ipv6Addr()
	}

	activeFrom, activeTo := devicePresenceWindow(ctx, mdls, deviceID)

	return fingerprint.MergeCandidate{
		DeviceID:     deviceID,
		Fingerprints: fingerprint.ToStoredFingerprints(records),
		LastActivity: lastActivity,
		CurrentMAC:   dev.MacAddr(),
		CurrentIP:    currentIP,
		Hostname:     dev.Hostname(),
		ActiveFrom:   activeFrom,
		ActiveTo:     activeTo,
	}, true
}

// devicePresenceWindow returns [earliest first_seen_at, latest last_seen_at] across
// all of a device's device_macs rows — the span over which the device was observed on
// the network. Returns zero times when the device has no timestamped MAC rows, which
// disables the concurrency check for that candidate.
func devicePresenceWindow(ctx context.Context, mdls *models.Models, deviceID int64) (from, to time.Time) {
	macs, err := mdls.DeviceMac().FindByDeviceID(ctx, deviceID)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	for _, m := range macs {
		if m.FirstSeenAt.Valid && (from.IsZero() || m.FirstSeenAt.Time.Before(from)) {
			from = m.FirstSeenAt.Time
		}
		if m.LastSeenAt.Valid && m.LastSeenAt.Time.After(to) {
			to = m.LastSeenAt.Time
		}
	}
	return from, to
}

// hasRunningSession reports whether the device currently has an active session.
func hasRunningSession(ctx context.Context, clientMgr *sessmgr.SessionsMgr, deviceID int64) bool {
	clnt, err := clientMgr.FindDeviceByID(ctx, deviceID)
	if err != nil || clnt == nil {
		return false
	}
	_, ok := clientMgr.GetRunningSession(clnt)
	return ok
}
