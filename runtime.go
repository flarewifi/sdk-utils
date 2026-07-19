package sdkutils

import (
	"runtime"
	"strings"
)

var (
	GOOS             string
	GO_SHORT_VERSION string // e.g. "1.20"
	// GO_VERSION is the running binary's own Go (runtime.Version, e.g. "1.20.5").
	// For builder bots this is now only the bot's NATIVE/default Go — one element of
	// its build capability set (HostBuildableGoVersions) — and is NO LONGER the
	// task-claim key. A release is claimed by matching its authoritative
	// software_releases.go_version against the host's capability set, so a bot can
	// build a release whose pinned Go differs from this value.
	GO_VERSION      string
	GO_LONG_VERSION string // e.g. "go1.20.5"
	GOARCH          string
)

func init() {
	v := runtime.Version()
	GO_VERSION = strings.Replace(v, "go", "", 1)
	varr := strings.Split(GO_VERSION, ".")
	GO_SHORT_VERSION = varr[0] + "." + varr[1]
	GOARCH = runtime.GOARCH
	GOOS = runtime.GOOS
}
