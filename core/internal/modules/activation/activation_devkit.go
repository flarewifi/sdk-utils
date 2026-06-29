//go:build devkit

package activation

// Devkit builds never register with or validate against the cloud. The machine is
// treated as permanently activated so the admin UI is reachable without an
// activation token and without contacting any server. See devkitBypass(), which
// short-circuits Validate().
func init() {
	IsActivated.Store(true)
}

// devkitBypass reports whether cloud activation must be skipped entirely.
func devkitBypass() bool {
	return true
}
