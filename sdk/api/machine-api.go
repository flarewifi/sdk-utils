package sdkapi

// IMachineApi provides machine-related information.
type IMachineApi interface {
	GetID() string

	// IsOnline reports whether the machine currently has internet access, as
	// observed by the core's online monitor. It reflects the same signal that
	// drives EventInternetUp / EventInternetDown (see IEventsApi.OnInternetEvent):
	// a periodic connectivity probe. It returns false before the first probe
	// completes, i.e. connectivity is treated as "not known to be up". For a
	// push-based reaction to connectivity changes, prefer OnInternetEvent over
	// polling this.
	IsOnline() bool
}
