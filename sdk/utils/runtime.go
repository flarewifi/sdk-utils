package sdkutils

import (
	"runtime"
	"strings"
)

var (
	GOOS             string
	GO_VERSION       string
	GO_LONG_VERSION  string
	GO_SHORT_VERSION string
	GOARCH           string
)

func init() {
	v := runtime.Version()
	GO_VERSION = strings.Replace(v, "go", "", 1)
	varr := strings.Split(GO_VERSION, ".")
	GO_SHORT_VERSION = varr[0] + "." + varr[1]
	GOARCH = runtime.GOARCH
	GOOS = runtime.GOOS
}
