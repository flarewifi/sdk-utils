package sdkutils

import "fmt"

// FormatByteData formats data in megabytes as a string, using GB if >= 1024 MB
func FormatByteData(dataMB float64) string {
	if dataMB >= 1024 {
		return fmt.Sprintf("%.1fG", dataMB/1024)
	}
	return fmt.Sprintf("%.2f MB", dataMB)
}

// FormatTimeSecs formats seconds into days:hours:minutes:seconds string
func FormatTimeSecs(timeSec int) string {
	days := timeSec / 86400
	timeSec %= 86400
	hours := timeSec / 3600
	timeSec %= 3600
	minutes := timeSec / 60
	seconds := timeSec % 60
	return fmt.Sprintf("%dd:%dh:%dm:%ds", days, hours, minutes, seconds)
}
