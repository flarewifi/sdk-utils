package sdkapi

// IMachineApi provides machine-related information.
type IMachineApi interface {
	GetID() string

	// ProductVersion returns the machine's per-B2B-partner product version — the
	// operator-set release version this build was stamped with (core/product.json),
	// which the machine reports for software-update eligibility. It is distinct from
	// the core version (plugin.json "version", the ABI identity): a partner's
	// product lineage advances independently of the underlying core. Falls back to
	// the core version on builds that were never stamped (older images / dev).
	ProductVersion() string

	// IsOnline reports whether the machine currently has internet access, as
	// observed by the core's online monitor. It reflects the same signal that
	// drives EventInternetUp / EventInternetDown (see IEventsApi.OnInternetEvent):
	// a periodic connectivity probe. It returns false before the first probe
	// completes, i.e. connectivity is treated as "not known to be up". For a
	// push-based reaction to connectivity changes, prefer OnInternetEvent over
	// polling this.
	IsOnline() bool
}
