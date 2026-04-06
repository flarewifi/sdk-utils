//go:build !dev

package jobs

import "time"

// Production mode: Jobs run at specific times daily

var (
	// FingerprintCleanupInterval - runs daily at 3:00 AM (calculated dynamically)
	// Set to 0 to use time-of-day scheduling instead of interval
	FingerprintCleanupInterval = time.Duration(0)

	// LogCleanupInterval - runs every hour
	LogCleanupInterval = 1 * time.Hour

	// MachinePingInterval - pings server every hour
	MachinePingInterval = 1 * time.Hour

	// MachinePingInitialDelay - delay before first ping after startup
	MachinePingInitialDelay = 30 * time.Second

	// SessionCleanupInterval - runs daily at 23:30 (calculated dynamically)
	// Set to 0 to use time-of-day scheduling instead of interval
	SessionCleanupInterval = time.Duration(0)

	// BatchSaveInterval controls how often the batch save loop snapshots
	// time consumption for all running sessions and persists them to the
	// database in a single transaction.
	BatchSaveInterval = 1 * time.Minute
)

// Production schedule times (hour, minute)
const (
	FingerprintCleanupHour   = 3
	FingerprintCleanupMinute = 0

	SessionCleanupHour   = 23
	SessionCleanupMinute = 30

	// MaxFingerprintsPerDevice - maximum fingerprints to keep per device
	// Older fingerprints beyond this limit are deleted during cleanup
	MaxFingerprintsPerDevice = 10
)
