package fingerprint

import (
	"fmt"
)

func TestMergeScenario() {
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

	fmt.Printf("Validation result: %d\n", result)
	fmt.Printf("NoMatch=0, SmartMatch=1, ExactMatch=2\n")
	
	if result == SmartMatch {
		fmt.Println("✓ SmartMatch: Same OS+screen+lang (valid for identification, NOT for merge)")
	} else if result == ExactMatch {
		fmt.Println("✓ EXACT MATCH: Devices should merge!")
	} else {
		fmt.Println("✗ NoMatch: Devices will NOT merge")
	}
}
