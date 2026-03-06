//go:build dev

package jobs

import "time"

// Dev mode: All scheduled jobs run every 5 seconds for immediate testing

var (
	// FingerprintCleanupInterval - how often to run fingerprint cleanup
	FingerprintCleanupInterval = 5 * time.Second

	// DeviceMergeInterval - how often to run device merge
	DeviceMergeInterval = 5 * time.Second

	// LogCleanupInterval - how often to run log cleanup
	LogCleanupInterval = 5 * time.Second

	// MachinePingInterval - how often to ping the server
	MachinePingInterval = 5 * time.Second

	// MachinePingInitialDelay - delay before first ping after startup
	MachinePingInitialDelay = 2 * time.Second
)

// Schedule times (not used in dev mode, but needed for compilation)
const (
	FingerprintCleanupHour   = 3
	FingerprintCleanupMinute = 0

	DeviceMergeHour   = 3
	DeviceMergeMinute = 30

	// MaxFingerprintsPerDevice - maximum fingerprints to keep per device
	// In dev mode, use same value as production
	MaxFingerprintsPerDevice = 10
)
