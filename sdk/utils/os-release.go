package sdkutils

// OsRelease is the true immutable OS-image identity, written ONCE at OS-image
// build/flash time (SaveOsReleaseInfo) and never rewritten by an OTA update.
// DeviceModel lives here (moved back from core/product.json) because it must
// stay stable for the device's physical lifetime — unlike brand_id/device_config,
// which restamp on every software-release build to track update-eligibility.
type OsRelease struct {
	Os          string `json:"os"`
	OsVersion   string `json:"os_version"`
	OsTarget    string `json:"os_target"`
	OsArch      string `json:"os_arch"`
	OsProfile   string `json:"os_profile"`
	DeviceModel string `json:"device_model"`
}

const OsReleaseFile = "os_release.json"

func ReadOsRelease(file string) (OsRelease, error) {
	var release OsRelease
	err := JsonRead(file, &release)
	return release, err
}

func WriteOsRelease(file string, release OsRelease) error {
	return JsonWrite(file, &release)
}
