package sdkutils

import (
	"fmt"
	"sync"
	"time"
)

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

// UtcToLocalTime converts a UTC time to the server's local timezone.
func UtcToLocalTime(t time.Time) time.Time {
	return t.In(time.Local)
}

var (
	tzLocCache   = map[string]*time.Location{}
	tzLocCacheMu sync.RWMutex
)

// UtcToLocal converts a UTC time to the given IANA timezone (e.g.
// "Asia/Manila"), falling back to the server process's own local zone
// (time.Local) when timezone is empty or unrecognized -- callers pass in
// AppConfig.Timezone (application.json's configured zone) rather than
// trusting the OS/container's local zone, which can be misconfigured or
// unavailable independent of the application's own settings.
//
// Resolved *time.Location values are cached by zone name: time.LoadLocation
// reads zoneinfo data from disk, and this is meant to be called once per
// timestamp rendered, including in list views with many rows.
func UtcToLocal(t time.Time, timezone string) time.Time {
	return t.In(cachedLocation(timezone))
}

func cachedLocation(timezone string) *time.Location {
	if timezone == "" {
		return time.Local
	}

	tzLocCacheMu.RLock()
	loc, ok := tzLocCache[timezone]
	tzLocCacheMu.RUnlock()
	if ok {
		return loc
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Local
	}

	tzLocCacheMu.Lock()
	tzLocCache[timezone] = loc
	tzLocCacheMu.Unlock()

	return loc
}
