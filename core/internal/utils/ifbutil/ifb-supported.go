package ifbutil

import (
	"sync/atomic"

	cmd "tools/shell"
)

var (
	supported atomic.Bool
)

func init() {
	err := cmd.Exec("modprobe ifb", nil)
	supported.Store(err == nil)
}

// check if ifb interface is supported
func IsIfbSupported() bool {
	return supported.Load()
}
