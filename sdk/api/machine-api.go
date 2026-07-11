package sdkapi

// SystemStats holds a point-in-time snapshot of the machine's system resources.
type SystemStats struct {
	CpuPercent float64 // average CPU utilization across all cores, 0-100
	MemTotal   uint64  // bytes
	MemUsed    uint64  // bytes
	DiskTotal  uint64  // bytes
	DiskUsed   uint64  // bytes

	// TemperatureCelsius is nil when the machine exposes no readable thermal
	// sensor (common on many OpenWRT devices) rather than a misleading zero.
	TemperatureCelsius *float64
}

// IMachineApi provides machine-related information.
type IMachineApi interface {
	GetID() string

	// SystemStats returns a snapshot of the machine's current CPU, memory, disk,
	// and temperature usage.
	SystemStats() SystemStats

	// ProductVersion returns the machine's per-B2B-partner product version — the
	// operator-set release version this build was stamped with (core/product.json),
	// which the machine reports for software-update eligibility. It is distinct from
	// the core version (plugin.json "version", the ABI identity): a partner's
	// product lineage advances independently of the underlying core. Falls back to
	// the core version on builds that were never stamped (older images / dev).
	ProductVersion() string

	// DeviceModel returns the machine's board/device model (e.g. "z-router-2660"),
	// read from the frozen /etc/os_release.json — stamped once at OS-image
	// build/flash time and never rewritten by an OTA, unlike ProductVersion's
	// core/product.json. Returns "" if unreadable (e.g. local dev).
	DeviceModel() string

	// IsOnline reports whether the machine currently has internet access, as
	// observed by the core's online monitor. It reflects the same signal that
	// drives EventInternetUp / EventInternetDown (see IEventsApi.OnInternetEvent):
	// a periodic connectivity probe. It returns false before the first probe
	// completes, i.e. connectivity is treated as "not known to be up". For a
	// push-based reaction to connectivity changes, prefer OnInternetEvent over
	// polling this.
	IsOnline() bool
}
