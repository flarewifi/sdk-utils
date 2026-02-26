package utils

// toFloat64 converts an interface{} value from SQLite (which can be int64 or float64) to float64.
func ToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}
