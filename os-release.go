package sdkutils

type OsRelease struct {
	DeviceModel  string `json:"device_model"`
	DeviceConfig string `json:"device_config"`
	BrandId      string `json:"brand_id"`
	Os           string `json:"os"`
	OsVersion    string `json:"os_version"`
	OsTarget     string `json:"os_target"`
	OsArch       string `json:"os_arch"`
	OsProfile    string `json:"os_profile"`
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
