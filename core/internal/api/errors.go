package api

import (
	"database/sql"
	"errors"
	"strings"

	sdkapi "sdk/api"
)

// SanitizeError converts internal errors to user-safe messages
// This prevents database schema leakage and other security issues
func SanitizeError(api sdkapi.IPluginApi, err error) (userMsg string, status int) {
	// Check for known error types
	if errors.Is(err, sql.ErrNoRows) {
		return api.Translate("error", "Resource not found"), 404
	}

	// Check for database errors (don't expose schema)
	if isDatabaseError(err) {
		return api.Translate("error", "A database error occurred"), 500
	}

	// Check for network/connectivity errors
	if isNetworkError(err) {
		return api.Translate("error", "Network error occurred"), 500
	}

	// Default: hide all internal details
	return api.Translate("error", "An unexpected error occurred"), 500
}

func isDatabaseError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "pq:") ||
		strings.Contains(errStr, "sqlite") ||
		strings.Contains(errStr, "database") ||
		strings.Contains(errStr, "sql") ||
		strings.Contains(errStr, "constraint") ||
		strings.Contains(errStr, "duplicate key")
}

func isNetworkError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network")
}
