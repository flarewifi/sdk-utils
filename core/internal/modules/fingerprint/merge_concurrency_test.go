package fingerprint

import (
	"testing"
	"time"
)

// exactMatchFingerprint is a single fingerprint two candidates can share so that
// FingerprintsMatchKind returns MatchBrowser (ExactMatch) — the case where the only
// thing standing between a correct dedup and a wrong merge is the concurrency guard.
func exactMatchFingerprint() []StoredFingerprint {
	return []StoredFingerprint{{
		FingerprintHash:  "b34b4bb609ab1184bfa2a94672a828b900309ccb0e1a9261fb1cde12c6629cd2",
		OSFamily:         "Android",
		ScreenResolution: "360x770",
		Language:         "en-US",
		Timezone:         "UTC+8",
		IsCna:            false,
	}}
}

func TestShouldMergeDevices_ConcurrentPresence_DoesNotMerge(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	// Two same-model phones, identical fingerprint, that were both on the network
	// across an overlapping multi-hour window — impossible for one physical device.
	a := MergeCandidate{
		DeviceID:     1,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base,
		ActiveTo:     base.Add(3 * time.Hour),
	}
	b := MergeCandidate{
		DeviceID:     2,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base.Add(1 * time.Hour), // overlaps a's window by ~2h
		ActiveTo:     base.Add(4 * time.Hour),
	}

	if d := ShouldMergeDevices(a, b); d.ShouldMerge {
		t.Fatalf("expected no merge for concurrently-active distinct devices, got merge (reason=%q)", d.Reason)
	}
}

func TestShouldMergeDevices_SequentialHandoff_Merges(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	// Same physical device: old row goes quiet, new row picks up seconds later
	// (identity split from a lost cookie). Windows are adjacent, not overlapping.
	old := MergeCandidate{
		DeviceID:     1,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base,
		ActiveTo:     base.Add(2 * time.Hour),
	}
	fresh := MergeCandidate{
		DeviceID:     2,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base.Add(2*time.Hour + 30*time.Second), // starts after old ends
		ActiveTo:     base.Add(5 * time.Hour),
	}

	if d := ShouldMergeDevices(old, fresh); !d.ShouldMerge {
		t.Fatalf("expected merge for sequential handoff (same device), got no merge (reason=%q)", d.Reason)
	}
}

func TestShouldMergeDevices_OverlapWithinTolerance_Merges(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	// A tiny transition overlap (under ConcurrencyTolerance) must not block a merge.
	old := MergeCandidate{
		DeviceID:     1,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base,
		ActiveTo:     base.Add(2 * time.Hour),
	}
	fresh := MergeCandidate{
		DeviceID:     2,
		Fingerprints: exactMatchFingerprint(),
		ActiveFrom:   base.Add(2*time.Hour - ConcurrencyTolerance/2), // overlaps by tolerance/2
		ActiveTo:     base.Add(5 * time.Hour),
	}

	if d := ShouldMergeDevices(old, fresh); !d.ShouldMerge {
		t.Fatalf("expected merge when overlap is within tolerance, got no merge (reason=%q)", d.Reason)
	}
}

func TestShouldMergeDevices_UnknownWindows_MergesOnFingerprint(t *testing.T) {
	// No presence windows (zero times) — concurrency cannot be assessed, so the
	// fingerprint decision stands (preserves pre-guard behavior).
	a := MergeCandidate{DeviceID: 1, Fingerprints: exactMatchFingerprint()}
	b := MergeCandidate{DeviceID: 2, Fingerprints: exactMatchFingerprint()}

	if d := ShouldMergeDevices(a, b); !d.ShouldMerge {
		t.Fatalf("expected merge when windows are unknown, got no merge (reason=%q)", d.Reason)
	}
}
