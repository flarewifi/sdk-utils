package sdkutils

import "fmt"

func FormatDuration(secs int) string {
	days := secs / (24 * 3600)
	secs %= 24 * 3600

	hours := secs / 3600
	secs %= 3600

	minutes := secs / 60
	secs %= 60

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dm ", minutes)
	}
	if secs > 0 || result == "" {
		result += fmt.Sprintf("%ds", secs)
	}

	return result
}
