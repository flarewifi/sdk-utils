package env

const (
	ENV_DEV int8 = iota
	ENV_SANDBOX
	ENV_STAGING
	ENV_PRODUCTION
)

// IsDevEnv reports whether this is a local development build (ENV_DEV). Use it to
// gate behaviour that must be STRICT and LOUD during active development but
// RESILIENT on any DEPLOYED device (staging, sandbox, production).
//
// The motivating case is plugin compile/load failure at boot. In dev a broken or
// ABI-stale plugin must abort the boot (os.Exit) so the failure is obvious in
// `flare server` logs / reflex and gets fixed. On a deployed device it must NOT
// abort: the boot script (start.sh) treats a non-zero exit as a crash and rolls
// the ENTIRE staged software update back, so one un-rebuilt plugin would silently
// revert an otherwise-good update. A deployed device instead notifies, recovers
// from backup if possible, and keeps booting with that plugin absent. Staging is a
// deployed device — it must behave like production here, not like dev.
func IsDevEnv() bool {
	return GO_ENV == ENV_DEV
}

// GoEnvString returns the current build environment as a lowercase string
// ("development", "sandbox", "staging", "production"). It is exposed to plugin
// install scripts via the GO_ENV environment variable so they can behave per
// environment (e.g. skip device-only setup in development).
func GoEnvString() string {
	switch GO_ENV {
	case ENV_DEV:
		return "development"
	case ENV_SANDBOX:
		return "sandbox"
	case ENV_STAGING:
		return "staging"
	case ENV_PRODUCTION:
		return "production"
	}
	return "unknown"
}
