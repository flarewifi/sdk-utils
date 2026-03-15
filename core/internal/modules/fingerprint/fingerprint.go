package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type FingerprintData struct {
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

// normalizeScreenResolution normalizes screen resolution by ensuring width <= height
// This treats portrait and landscape orientations as equivalent (e.g., 360x800 and 800x360)
func normalizeScreenResolution(res string) string {
	if res == "" {
		return ""
	}

	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return res // Invalid format, return as-is
	}

	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return res // Parse error, return as-is
	}

	// Always return smaller dimension first (portrait orientation)
	if w > h {
		return fmt.Sprintf("%dx%d", h, w)
	}
	return res
}

// extractPrimaryLanguage extracts the primary language code from a locale string
// Examples: "en-US" -> "en", "en-GB" -> "en", "fr-FR" -> "fr", "en" -> "en"
func extractPrimaryLanguage(lang string) string {
	if lang == "" {
		return ""
	}
	// Split on hyphen and take the first part
	parts := strings.Split(lang, "-")
	return strings.ToLower(strings.TrimSpace(parts[0]))
}

// GenerateHash creates SHA256 hash from fingerprint data
// Screen resolution is normalized to handle device rotation
func GenerateHash(data FingerprintData) string {
	normalizedScreen := normalizeScreenResolution(data.ScreenRes)
	combined := fmt.Sprintf("%s|%s|%s|%s",
		data.UserAgent,
		normalizedScreen,
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
	// Smart match: Same OS + Screen + Primary Language (+ Timezone if both available)
	// This allows multiple browsers per device and language locale changes while maintaining security
	if !stored.IsCna && !currentIsCNA {
		// Normalize screen resolutions for comparison (handles device rotation)
		normalizedStoredScreen := normalizeScreenResolution(stored.ScreenResolution)
		normalizedCurrentScreen := normalizeScreenResolution(currentScreen)

		// Extract primary language codes for comparison (e.g., "en-US" -> "en", "en-GB" -> "en")
		// This allows language locale changes (en-US <-> en-GB) without breaking validation
		storedPrimaryLang := extractPrimaryLanguage(stored.Language)
		currentPrimaryLang := extractPrimaryLanguage(currentLang)

		// Base validation: OS + Screen + Primary Language must match
		if stored.OSFamily != "" && stored.OSFamily == currentOS &&
			normalizedStoredScreen != "" && normalizedStoredScreen == normalizedCurrentScreen &&
			storedPrimaryLang != "" && storedPrimaryLang == currentPrimaryLang {

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
