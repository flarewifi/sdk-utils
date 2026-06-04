package fingerprint

import (
	"time"
)

// MatchKind describes how two fingerprint sets matched.
type MatchKind int

const (
	MatchNone    MatchKind = iota // fingerprints do not match
	MatchBrowser                  // ExactMatch or browser SmartMatch (OS+screen+lang)
	MatchCNA                      // CNA-CNA SmartMatch (OS-family only — weaker signal)
)

// MergeCandidate represents a device being evaluated for merging.
type MergeCandidate struct {
	DeviceID     int64
	Fingerprints []StoredFingerprint
	LastActivity time.Time
	CurrentMAC   string // current MAC address (is_current=TRUE in device_macs)
	CurrentIP    string // current IP address (devices.ip_address)
	Hostname     string // hostname (devices.hostname), may be empty
}

// MergeDecision contains the result of evaluating two devices for merging.
type MergeDecision struct {
	ShouldMerge bool
	TargetID    int64  // Device to keep
	SourceID    int64  // Device to merge into target
	Reason      string // Explanation for the decision
}

// ShouldMergeDevices determines if two devices should be merged based on fingerprints.
//
// Rules:
//   - Both devices must have fingerprints — without fingerprints on both sides we cannot
//     verify they are the same physical device.
//   - If fingerprints match (ExactMatch or browser SmartMatch): merge.
//   - For CNA-CNA SmartMatch (OS-family only): additionally require same MAC, IP, and
//     hostname to guard against false positives.
//   - The device with most recent session activity is kept as the target.
func ShouldMergeDevices(deviceA, deviceB MergeCandidate) MergeDecision {
	hasA := len(deviceA.Fingerprints) > 0
	hasB := len(deviceB.Fingerprints) > 0

	// Case 1: Both have fingerprints - only merge if they match
	if hasA && hasB {
		kind := FingerprintsMatchKind(deviceA.Fingerprints, deviceB.Fingerprints)
		if kind == MatchNone {
			return MergeDecision{ShouldMerge: false}
		}

		// CNA-CNA SmartMatch is OS-family-only — require same current MAC and IP
		// (and hostname when both non-empty) to guard against false positives.
		if kind == MatchCNA && !CnaMACIPMatch(deviceA, deviceB) {
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

	// Case 2, 3, 4: At least one device has no fingerprints — skip merge.
	// Without fingerprints on both sides we cannot verify they are the same
	// physical device. Shared MAC alone is not sufficient (MAC randomization,
	// reuse across devices). Devices will be reconsidered once both accumulate
	// browser fingerprints through normal registration.
	return MergeDecision{ShouldMerge: false}
}

// FingerprintsMatchKind checks if two devices have matching fingerprints and returns
// the kind of match found.
//
// CNA guard: when either fingerprint is a CNA fingerprint, ValidateFingerprint
// may return SmartMatch based solely on OS family — which is too permissive for
// a destructive merge operation. In that case we require either an ExactMatch
// (hash collision unlikely but correct) or that BOTH fingerprints are CNA with
// the same OS family (same device class, acceptable risk — caller must additionally
// verify MAC+IP identity via CnaMACIPMatch).
func FingerprintsMatchKind(fpsA, fpsB []StoredFingerprint) MatchKind {
	best := MatchNone
	for _, a := range fpsA {
		for _, b := range fpsB {
			result := ValidateFingerprint(
				a,
				b.FingerprintHash,
				b.OSFamily,
				b.ScreenResolution,
				b.Language,
				b.Timezone,
				b.IsCna,
			)

			if result == ExactMatch {
				return MatchBrowser // ExactMatch is always definitive
			}

			if result == SmartMatch {
				if a.IsCna && b.IsCna {
					// CNA-CNA SmartMatch: OS-family-only — weaker signal.
					// Caller must verify MAC+IP before acting on this.
					if best < MatchCNA {
						best = MatchCNA
					}
				} else if !a.IsCna && !b.IsCna {
					// Browser SmartMatch (OS + screen + language) — too loose for merge.
					// Two different Android devices of the same model/locale would match.
					// Only ExactMatch (full hash) is accepted for browser-to-browser merges.
				} else {
					// One CNA, one browser: SmartMatch is OS-only — too weak for a merge.
				}
			}
		}
	}
	return best
}

// CnaMACIPMatch returns true when two CNA-matched candidates share the same
// current MAC address and IP address. If both devices have a non-empty hostname,
// those must also match.
//
// An empty MAC or IP on either side causes an automatic false — a device with no
// current MAC/IP record cannot be safely merged on CNA evidence alone.
func CnaMACIPMatch(a, b MergeCandidate) bool {
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

// ToStoredFingerprints converts a slice of fingerprint data (from any source)
// into StoredFingerprint values suitable for merge comparison.
// This helper avoids duplicating the conversion logic across callers.
type FingerprintRecord struct {
	FingerprintHash  string
	OsFamily         string
	ScreenResolution string
	Language         string
	Timezone         string
	IsCna            bool
}

func ToStoredFingerprints(records []FingerprintRecord) []StoredFingerprint {
	result := make([]StoredFingerprint, len(records))
	for i, r := range records {
		result[i] = StoredFingerprint{
			FingerprintHash:  r.FingerprintHash,
			OSFamily:         r.OsFamily,
			ScreenResolution: r.ScreenResolution,
			Language:         r.Language,
			Timezone:         r.Timezone,
			IsCna:            r.IsCna,
		}
	}
	return result
}
