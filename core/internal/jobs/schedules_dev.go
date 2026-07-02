//go:build dev

package jobs

import "time"

// Dev mode: All scheduled jobs run every 5 seconds for immediate testing

var (
	// FingerprintCleanupInterval - how often to run fingerprint cleanup
	FingerprintCleanupInterval = 500 * time.Second

	// LogCleanupInterval - how often to run log cleanup
	LogCleanupInterval = 500 * time.Second

	// MachinePingInterval - how often to ping the server
	MachinePingInterval = 500 * time.Second

	// MachinePingInitialDelay - delay before first ping after startup
	MachinePingInitialDelay = 200 * time.Second

	// PortalCertInterval - how often to check the cloud for a renewed portal cert
	PortalCertInterval = 600 * time.Second

	// PortalCertInitialDelay - delay before the first portal cert fetch after startup
	PortalCertInitialDelay = 15 * time.Second

	// BlockedPluginsInterval - how often to poll the cloud denylist (fast in dev)
	BlockedPluginsInterval = 600 * time.Second

	// BlockedPluginsInitialDelay - delay before the first denylist fetch after startup
	BlockedPluginsInitialDelay = 20 * time.Second

	// InstalledPluginsReportInterval - daily backstop; install/uninstall hooks
	// trigger an immediate report, so the periodic tick only needs to be daily.
	InstalledPluginsReportInterval = 24 * time.Hour

	// InstalledPluginsReportInitialDelay - delay before the first installed-plugins report after startup
	InstalledPluginsReportInitialDelay = 25 * time.Second

	// SessionCleanupInterval - how often to run session cleanup
	SessionCleanupInterval = 500 * time.Second

	// NotificationCleanupInterval - how often to run notification cleanup
	NotificationCleanupInterval = 500 * time.Second

	// VoucherCleanupInterval - how often to run voucher cleanup
	VoucherCleanupInterval = 500 * time.Second

	// BatchSaveInterval controls how often the batch save loop snapshots
	// time consumption for all running sessions and persists them to the
	// database in a single transaction.
	BatchSaveInterval = 30 * time.Second

	// DeviceMergeInterval - how often to run the device-merge reconciliation job (fast in dev)
	DeviceMergeInterval = 300 * time.Second

	// DeviceMergeInitialDelay - delay before the first reconciliation pass after startup
	DeviceMergeInitialDelay = 45 * time.Second
)

// Schedule times (not used in dev mode, but needed for compilation)
const (
	FingerprintCleanupHour   = 3
	FingerprintCleanupMinute = 0

	SessionCleanupHour   = 23
	SessionCleanupMinute = 30

	LogCleanupHour   = 1
	LogCleanupMinute = 0

	NotificationCleanupHour   = 2
	NotificationCleanupMinute = 0

	VoucherCleanupHour   = 2
	VoucherCleanupMinute = 15

	// MaxFingerprintsPerDevice - maximum fingerprints to keep per device
	// In dev mode, use same value as production
	MaxFingerprintsPerDevice = 10
)
