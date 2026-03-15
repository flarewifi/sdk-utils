package fingerprint

import (
	"testing"
)

func TestMergeScenarioDevice8And34(t *testing.T) {
	// Device 8 fingerprint (en-TR)
	stored := StoredFingerprint{
		FingerprintHash:  "b34b4bb609ab1184bfa2a94672a828b900309ccb0e1a9261fb1cde12c6629cd2",
		OSFamily:         "Android",
		ScreenResolution: "360x770",
		Language:         "en-TR",
		Timezone:         "UTC+8",
		IsCna:            false,
	}

	// Device 34 fingerprint (en-GB)
	currentHash := "8a8371c3ae8c5d9c0f2b902abc3a299a9b788e32c9e88e684f04a2167a6804c1"
	currentOS := "Android"
	currentScreen := "360x770"
	currentLang := "en-GB"
	currentTZ := "UTC+8"
	currentIsCNA := false

	result := ValidateFingerprint(stored, currentHash, currentOS, currentScreen, currentLang, currentTZ, currentIsCNA)

	if result != SmartMatch {
		t.Errorf("Devices 8 and 34 should SmartMatch (en-TR vs en-GB are both 'en'), got %v", result)
	}
}
