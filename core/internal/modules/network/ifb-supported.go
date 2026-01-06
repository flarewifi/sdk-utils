package networkutil

import (
	"sync/atomic"

	cmd "core/utils/shell"
)

var (
	ifbSupported atomic.Bool
)

func init() {
	err := cmd.Exec("modprobe ifb", nil)
	ifbSupported.Store(err == nil)
}

// check if ifb interface is supported
func IsIfbSupported() bool {
	return ifbSupported.Load()
}
