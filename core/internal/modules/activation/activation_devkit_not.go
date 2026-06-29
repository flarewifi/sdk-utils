//go:build !devkit

package activation

// devkitBypass reports whether cloud activation must be skipped entirely. Always
// false in non-devkit builds — normal activation/validation applies.
func devkitBypass() bool {
	return false
}
