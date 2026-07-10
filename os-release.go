package sdkutils

// OsRelease is the true immutable OS-image identity, written ONCE at OS-image
// build/flash time (SaveOsReleaseInfo) and never rewritten by an OTA update.
// brand_id/device_model/device_config used to live here too, but they've moved
// to core/product.json (restamped, encrypted, on every software-release build)
// since update-eligibility/product-transfer matching needs them to track the
// machine's CURRENTLY installed release, not what it was originally flashed with.
type OsRelease struct {
	Os        string `json:"os"`
	OsVersion string `json:"os_version"`
	OsTarget  string `json:"os_target"`
	OsArch    string `json:"os_arch"`
	OsProfile string `json:"os_profile"`
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
