package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type FingerprintData struct {
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

// GenerateHash creates SHA256 hash from fingerprint data
func GenerateHash(data FingerprintData) string {
	combined := fmt.Sprintf("%s|%s|%s|%s",
		data.UserAgent,
		data.ScreenRes,
		data.Language,
		data.Timezone,
	)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

type ValidationResult int

const (
	NoMatch ValidationResult = iota
	SmartMatch
	ExactMatch
)

type StoredFingerprint struct {
	FingerprintHash  string
	OSFamily         string
	ScreenResolution string
	Language         string
	Timezone         string
}

// ValidateFingerprint checks if current fingerprint matches stored fingerprint
// Returns ExactMatch, SmartMatch, or NoMatch
// Smart match now requires: Same OS + Screen + Language + Timezone (4 data points for security)
func ValidateFingerprint(stored StoredFingerprint, currentHash string, currentOS string, currentScreen string, currentLang string, currentTZ string) ValidationResult {
	// Exact hash match
	if stored.FingerprintHash == currentHash {
		return ExactMatch
	}

	// Smart match: Same OS + Screen + Language + Timezone (handles browser switches)
	// Requires all 4 fields to match for better security against cookie theft
	if stored.OSFamily != "" && stored.OSFamily == currentOS &&
		stored.ScreenResolution != "" && stored.ScreenResolution == currentScreen &&
		stored.Language != "" && stored.Language == currentLang &&
		stored.Timezone != "" && stored.Timezone == currentTZ {
		return SmartMatch
	}

	return NoMatch
}
