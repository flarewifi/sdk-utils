package env

const (
	ENV_DEV int8 = iota
	ENV_SANDBOX
	ENV_STAGING
	ENV_PRODUCTION
)

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
