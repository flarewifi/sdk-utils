//go:build !mono

package logger

// Non-mono (dev/server) build: not flash-constrained, so keep a large on-disk log
// for easier debugging. See the comment in logger.go for the rotation contract.
const (
	maxLogBytes     = 100 << 20 // 100 MiB
	rotateKeepBytes = 50 << 20  // 50 MiB
)
