package fingerprint

import (
	"testing"
)

func TestNormalizeScreenResolution(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"360x800", "360x800"},     // Portrait (already normalized)
		{"800x360", "360x800"},     // Landscape (needs normalization)
		{"375x667", "375x667"},     // Portrait
		{"667x375", "375x667"},     // Landscape
		{"1080x1920", "1080x1920"}, // Portrait
		{"1920x1080", "1080x1920"}, // Landscape
		{"", ""},                   // Empty
		{"invalid", "invalid"},     // Invalid format
		{"100", "100"},             // Missing 'x'
	}

	for _, tc := range testCases {
		result := normalizeScreenResolution(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeScreenResolution(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestGenerateHashWithRotation(t *testing.T) {
	// Generate hashes for same device in portrait and landscape
	portraitData := FingerprintData{
		UserAgent: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36",
		ScreenRes: "360x800",
		Language:  "en-US",
		Timezone:  "UTC+8",
	}

	landscapeData := FingerprintData{
		UserAgent: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36",
		ScreenRes: "800x360", // Rotated screen
		Language:  "en-US",
		Timezone:  "UTC+8",
	}

	portraitHash := GenerateHash(portraitData)
	landscapeHash := GenerateHash(landscapeData)

	if portraitHash != landscapeHash {
		t.Errorf("Portrait and landscape should generate same hash:\nPortrait:  %s\nLandscape: %s", portraitHash, landscapeHash)
	}
}

func TestValidateFingerprintWithRotation(t *testing.T) {
	// Stored fingerprint in portrait orientation
	stored := StoredFingerprint{
		FingerprintHash:  "different-hash", // Force smart match path
		OSFamily:         "Android",
		ScreenResolution: "360x800",
		Language:         "en-US",
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	// Current fingerprint in landscape orientation
	result := ValidateFingerprint(stored, "test-hash", "Android", "800x360", "en-US", "UTC+8", false)

	if result != SmartMatch {
		t.Errorf("ValidateFingerprint with rotated screen should return SmartMatch, got %v", result)
	}
}

func TestValidateFingerprintExactMatch(t *testing.T) {
	// Test that exact hash match still works
	portraitData := FingerprintData{
		UserAgent: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36",
		ScreenRes: "360x800",
		Language:  "en-US",
		Timezone:  "UTC+8",
	}
	hash := GenerateHash(portraitData)

	stored := StoredFingerprint{
		FingerprintHash:  hash,
		OSFamily:         "Android",
		ScreenResolution: "360x800",
		Language:         "en-US",
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	// Test with same hash - should be ExactMatch
	result := ValidateFingerprint(stored, hash, "Android", "360x800", "en-US", "UTC+8", false)
	if result != ExactMatch {
		t.Errorf("ValidateFingerprint with exact hash should return ExactMatch, got %v", result)
	}

	// Test with rotated screen but same hash (because we normalize)
	result = ValidateFingerprint(stored, hash, "Android", "800x360", "en-US", "UTC+8", false)
	if result != ExactMatch {
		t.Errorf("ValidateFingerprint with exact hash (rotated screen) should return ExactMatch, got %v", result)
	}
}

func TestExtractPrimaryLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"US English", "en-US", "en"},
		{"British English", "en-GB", "en"},
		{"Turkish English", "en-TR", "en"},
		{"French France", "fr-FR", "fr"},
		{"Spanish Spain", "es-ES", "es"},
		{"Simple English", "en", "en"},
		{"Simple French", "fr", "fr"},
		{"Empty", "", ""},
		{"Lowercase already", "de-de", "de"},
		{"Mixed case", "En-US", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPrimaryLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("extractPrimaryLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateFingerprintLanguageVariation(t *testing.T) {
	// Test that language locale changes (en-US <-> en-GB) are accepted as SmartMatch
	tests := []struct {
		name           string
		storedLang     string
		currentLang    string
		expectedResult ValidationResult
	}{
		{"Same exact language", "en-US", "en-US", SmartMatch},
		{"Same primary language (US to GB)", "en-US", "en-GB", SmartMatch},
		{"Same primary language (GB to US)", "en-GB", "en-US", SmartMatch},
		{"Same primary language (GB to TR)", "en-GB", "en-TR", SmartMatch},
		{"Different language (en to fr)", "en-US", "fr-FR", NoMatch},
		{"Different language (en to es)", "en-GB", "es-ES", NoMatch},
		{"Same simple language", "en", "en", SmartMatch},
		{"Simple to locale (en to en-US)", "en", "en-US", SmartMatch},
		{"Locale to simple (en-GB to en)", "en-GB", "en", SmartMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := StoredFingerprint{
				FingerprintHash:  "different-hash", // Force smart match path
				OSFamily:         "Android",
				ScreenResolution: "360x800",
				Language:         tt.storedLang,
				Timezone:         "UTC+8",
				IsCna:            false,
			}

			result := ValidateFingerprint(stored, "test-hash", "Android", "360x800", tt.currentLang, "UTC+8", false)

			if result != tt.expectedResult {
				t.Errorf("ValidateFingerprint with stored=%q, current=%q should return %v, got %v",
					tt.storedLang, tt.currentLang, tt.expectedResult, result)
			}
		})
	}
}

func TestValidateFingerprintLanguageAndRotationCombined(t *testing.T) {
	// Test that both language variation AND screen rotation work together
	stored := StoredFingerprint{
		FingerprintHash:  "different-hash", // Force smart match path
		OSFamily:         "Android",
		ScreenResolution: "360x800", // Portrait
		Language:         "en-TR",   // Turkish locale
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	// Current: landscape orientation + different English locale
	result := ValidateFingerprint(stored, "test-hash", "Android", "800x360", "en-GB", "UTC+8", false)

	if result != SmartMatch {
		t.Errorf("ValidateFingerprint with rotated screen AND language locale change should return SmartMatch, got %v", result)
	}
}

func TestValidateFingerprintNoMatchDifferentOS(t *testing.T) {
	// Even with same language, different OS should not match
	stored := StoredFingerprint{
		FingerprintHash:  "different-hash",
		OSFamily:         "Android",
		ScreenResolution: "360x800",
		Language:         "en-US",
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	result := ValidateFingerprint(stored, "test-hash", "iOS", "360x800", "en-US", "UTC+8", false)

	if result != NoMatch {
		t.Errorf("ValidateFingerprint with different OS should return NoMatch, got %v", result)
	}
}

func TestValidateFingerprintNoMatchDifferentScreen(t *testing.T) {
	// Even with same language, different screen size should not match
	stored := StoredFingerprint{
		FingerprintHash:  "different-hash",
		OSFamily:         "Android",
		ScreenResolution: "360x800",
		Language:         "en-US",
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	result := ValidateFingerprint(stored, "test-hash", "Android", "375x667", "en-US", "UTC+8", false)

	if result != NoMatch {
		t.Errorf("ValidateFingerprint with different screen size should return NoMatch, got %v", result)
	}
}
