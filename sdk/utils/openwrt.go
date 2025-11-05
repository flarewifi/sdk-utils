package sdkutils

import (
	"os"
	"strings"
)

const (
	OpenWrtReleaseFile = "/etc/openwrt_release"
)

type OpenWrtRelease struct {
	DISTRIB_ID          string
	DISTRIB_RELEASE     string
	DISTRIB_REVISION    string
	DISTRIB_TARGET      string
	DISTRIB_ARCH        string
	DISTRIB_DESCRIPTION string
	DISTRIB_TAINTS      string
}

func ParseOpenWrtRelease() (release OpenWrtRelease, err error) {
	data, err := os.ReadFile(OpenWrtReleaseFile)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.SplitN(line, "=", 2)
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.Trim(fields[1], "'")
		switch key {
		case "DISTRIB_ID":
			release.DISTRIB_ID = value
		case "DISTRIB_RELEASE":
			release.DISTRIB_RELEASE = value
		case "DISTRIB_REVISION":
			release.DISTRIB_REVISION = value
		case "DISTRIB_TARGET":
			release.DISTRIB_TARGET = value
		case "DISTRIB_ARCH":
			release.DISTRIB_ARCH = value
		case "DISTRIB_DESCRIPTION":
			release.DISTRIB_DESCRIPTION = value
		case "DISTRIB_TAINTS":
			release.DISTRIB_TAINTS = value
		}
	}

	return
}
