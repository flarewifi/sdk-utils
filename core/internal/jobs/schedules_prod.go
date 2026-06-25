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

	// PortalCertInterval - how often to check the cloud for a renewed portal cert.
	// The cert is a 90-day Let's Encrypt cert renewed ~30 days before expiry, so a
	// daily check catches a renewal with weeks of slack. The fingerprint check is
	// cheap and reloads HTTPS only on an actual change; new/rebooted devices get
	// the cert promptly via PortalCertInitialDelay regardless of this interval.
	PortalCertInterval = 24 * time.Hour

	// PortalCertInitialDelay - delay before the first portal cert fetch after startup
	PortalCertInitialDelay = 60 * time.Second

	// BlockedPluginsInterval - how often to poll the cloud denylist of offending
	// plugins. Once a day is enough: a block only takes effect on the machine's
	// next reboot anyway (a loaded plugin .so cannot be unloaded mid-run), so a
	// tighter poll buys nothing. The initial fetch after boot catches plugins
	// flagged while the machine was offline.
	BlockedPluginsInterval = 24 * time.Hour

	// BlockedPluginsInitialDelay - delay before the first denylist fetch after startup
	BlockedPluginsInitialDelay = 90 * time.Second

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
