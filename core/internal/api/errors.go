package api

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	sdkapi "sdk/api"
)

// File path pattern to detect and sanitize
var filePathPattern = regexp.MustCompile(`(?i)(/[a-z0-9_\-\.]+)+\.[a-z]+|([a-z]:\\[^:*?"<>|\r\n]+)`)

// SanitizeError converts sensitive errors to user-safe messages
// Only RPC errors and file paths are sanitized - other errors pass through
func SanitizeError(api sdkapi.IPluginApi, err error) (userMsg string, status int) {
	// Check for known error types
	if errors.Is(err, sql.ErrNoRows) {
		return api.Translate("error", "Resource not found"), 404
	}

	errStr := err.Error()

	// Sanitize RPC/Twirp errors (may contain internal service details)
	if isRPCError(errStr) {
		return api.Translate("error", "Service temporarily unavailable"), 503
	}

	// Sanitize file path errors (may expose server structure)
	if containsFilePath(errStr) {
		return api.Translate("error", "A system error occurred"), 500
	}

	// All other errors pass through as-is (including database errors)
	return errStr, 500
}

// isRPCError checks if the error is from RPC/Twirp calls
func isRPCError(errStr string) bool {
	errLower := strings.ToLower(errStr)
	return strings.Contains(errLower, "twirp") ||
		strings.Contains(errLower, "rpc error") ||
		strings.Contains(errLower, "protobuf") ||
		strings.Contains(errLower, "grpc")
}

// containsFilePath checks if the error contains file system paths
func containsFilePath(errStr string) bool {
	// Check for common path patterns
	if filePathPattern.MatchString(errStr) {
		return true
	}

	// Check for common path-related error messages
	errLower := strings.ToLower(errStr)
	return strings.Contains(errLower, "no such file") ||
		strings.Contains(errLower, "permission denied") ||
		strings.Contains(errLower, "open /") ||
		strings.Contains(errLower, "stat /") ||
		strings.Contains(errLower, "read /") ||
		strings.Contains(errLower, "write /")
}
