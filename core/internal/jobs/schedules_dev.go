//go:build dev

package jobs

import "time"

// Dev mode: All scheduled jobs run every 5 seconds for immediate testing

var (
	// FingerprintCleanupInterval - how often to run fingerprint cleanup
	FingerprintCleanupInterval = 500 * time.Second

	// DeviceMergeInterval - how often to run device merge
	DeviceMergeInterval = 500 * time.Second

	// LogCleanupInterval - how often to run log cleanup
	LogCleanupInterval = 500 * time.Second

	// MachinePingInterval - how often to ping the server
	MachinePingInterval = 500 * time.Second

	// MachinePingInitialDelay - delay before first ping after startup
	MachinePingInitialDelay = 200 * time.Second

	// BatchSaveInterval controls how often the batch save loop snapshots
	// time consumption for all running sessions and persists them to the
	// database in a single transaction.
	BatchSaveInterval = 30 * time.Second
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
