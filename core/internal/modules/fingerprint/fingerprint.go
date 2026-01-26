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
	IsCna            bool // Indicates if this is a CNA fingerprint
}

// ValidateFingerprint checks if current fingerprint matches stored fingerprint
// Returns ExactMatch, SmartMatch, or NoMatch
// Handles CNA (partial fingerprints) and regular browser (full fingerprints)
func ValidateFingerprint(stored StoredFingerprint, currentHash string, currentOS string, currentScreen string, currentLang string, currentTZ string, currentIsCNA bool) ValidationResult {
	// Exact hash match
	if stored.FingerprintHash == currentHash {
		return ExactMatch
	}

	// CNA fingerprint handling
	// CNA fingerprints have empty Screen/Lang/TZ, so we match on OS only
	if stored.IsCna || currentIsCNA {
		if stored.OSFamily != "" && stored.OSFamily == currentOS {
			// Accept if same OS family
			// Covers: CNA->CNA, CNA->Browser, Browser->CNA
			return SmartMatch
		}
	}

	// Minimal fingerprint handling (JS disabled - User-Agent only, not CNA)
	// If current request has no Screen/Lang data but is not a CNA, match on OS only
	// This allows devices with JS disabled to be recognized
	if currentScreen == "" && currentLang == "" && !currentIsCNA {
		if stored.OSFamily != "" && stored.OSFamily == currentOS {
			// Accept if same OS family - JS disabled scenario
			return SmartMatch
		}
	}

	// Regular browser-to-browser smart match
	// Smart match: Same OS + Screen + Language (+ Timezone if both available)
	// This allows multiple browsers per device while maintaining security
	if !stored.IsCna && !currentIsCNA {
		// Base validation: OS + Screen + Language must match
		if stored.OSFamily != "" && stored.OSFamily == currentOS &&
			stored.ScreenResolution != "" && stored.ScreenResolution == currentScreen &&
			stored.Language != "" && stored.Language == currentLang {

			// Timezone validation (optional - only if both sides have it)
			// This provides backward compatibility for existing fingerprints without timezone
			if stored.Timezone != "" && currentTZ != "" {
				// Both have timezone - must match for security
				if stored.Timezone != currentTZ {
					return NoMatch // Different timezones - possible cookie theft
				}
			}
			// If either timezone is empty, skip TZ check (backward compatibility)

			return SmartMatch
		}
	}

	return NoMatch
}
