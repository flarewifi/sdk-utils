package sdkutils

import "fmt"

// FormatByteData formats data in megabytes as a string, using GB if >= 1024 MB
func FormatByteData(dataMB float64) string {
	if dataMB >= 1024 {
		return fmt.Sprintf("%.1fG", dataMB/1024)
	}
	return fmt.Sprintf("%.2f MB", dataMB)
}

// FormatTimeSecs formats seconds into a human-readable string
// Omits leading zeros for days, hours, and minutes, but ALWAYS shows seconds
// Examples: "5h 30m 0s", "2d 0h 3m 0s", "1d 0h 0m 30s", "45m 20s", "0s"
func FormatTimeSecs(timeSec int) string {
	days := timeSec / 86400
	timeSec %= 86400
	hours := timeSec / 3600
	timeSec %= 3600
	minutes := timeSec / 60
	seconds := timeSec % 60

	var result string
	started := false // Track if we've started adding components

	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
		started = true
	}
	if hours > 0 || (started && (minutes > 0 || seconds >= 0)) {
		result += fmt.Sprintf("%dh ", hours)
		started = true
	}
	if minutes > 0 || (started && seconds >= 0) {
		result += fmt.Sprintf("%dm ", minutes)
		started = true
	}
	// Always show seconds (never omit)
	result += fmt.Sprintf("%ds", seconds)

	return result
}
