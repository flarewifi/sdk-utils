package sdkutils

import "time"

// =============================================================================
// Pointer Creation Helpers
// =============================================================================

// IntPtr returns a pointer to the given int value.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to the given int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr returns a pointer to the given float64 value.
func Float64Ptr(f float64) *float64 {
	return &f
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// StringPtr returns a pointer to the given string value.
func StringPtr(s string) *string {
	return &s
}

// TimePtr returns a pointer to the given time.Time value.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// =============================================================================
// Pointer Copy Helpers (Deep Copy)
// =============================================================================

// CopyIntPtr creates a deep copy of an int pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyIntPtr(i *int) *int {
	if i == nil {
		return nil
	}
	copied := *i
	return &copied
}

// CopyInt64Ptr creates a deep copy of an int64 pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyInt64Ptr(i *int64) *int64 {
	if i == nil {
		return nil
	}
	copied := *i
	return &copied
}

// CopyFloat64Ptr creates a deep copy of a float64 pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyFloat64Ptr(f *float64) *float64 {
	if f == nil {
		return nil
	}
	copied := *f
	return &copied
}

// CopyBoolPtr creates a deep copy of a bool pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	copied := *b
	return &copied
}

// CopyStringPtr creates a deep copy of a string pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	copied := *s
	return &copied
}

// CopyTimePtr creates a deep copy of a time pointer to avoid shared state.
// Returns nil if the input is nil.
func CopyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copied := *t
	return &copied
}

// =============================================================================
// Pointer Equality Helpers
// =============================================================================

// IntPtrEqual compares two int pointers for equality.
// Returns true if both are nil, or both point to equal values.
func IntPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Int64PtrEqual compares two int64 pointers for equality.
// Returns true if both are nil, or both point to equal values.
func Int64PtrEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Float64PtrEqual compares two float64 pointers for equality.
// Returns true if both are nil, or both point to equal values.
func Float64PtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// BoolPtrEqual compares two bool pointers for equality.
// Returns true if both are nil, or both point to equal values.
func BoolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// StringPtrEqual compares two string pointers for equality.
// Returns true if both are nil, or both point to equal values.
func StringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// TimePtrEqual compares two time pointers for equality.
// Returns true if both are nil, or both point to equal times (using time.Equal).
func TimePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

// =============================================================================
// Value Extraction Helpers (with defaults)
// =============================================================================

// IntPtrVal returns the value of an int pointer, or the default if nil.
func IntPtrVal(p *int, defaultVal int) int {
	if p == nil {
		return defaultVal
	}
	return *p
}

// Int64PtrVal returns the value of an int64 pointer, or the default if nil.
func Int64PtrVal(p *int64, defaultVal int64) int64 {
	if p == nil {
		return defaultVal
	}
	return *p
}

// Float64PtrVal returns the value of a float64 pointer, or the default if nil.
func Float64PtrVal(p *float64, defaultVal float64) float64 {
	if p == nil {
		return defaultVal
	}
	return *p
}

// BoolPtrVal returns the value of a bool pointer, or the default if nil.
func BoolPtrVal(p *bool, defaultVal bool) bool {
	if p == nil {
		return defaultVal
	}
	return *p
}

// StringPtrVal returns the value of a string pointer, or the default if nil.
func StringPtrVal(p *string, defaultVal string) string {
	if p == nil {
		return defaultVal
	}
	return *p
}
