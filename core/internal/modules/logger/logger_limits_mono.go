//go:build mono

package logger

// Mono build: runs on routers with limited flash, so the on-disk log stays small.
// See the comment in logger.go for the rotation contract.
const (
	maxLogBytes     = 2 << 20 // 2 MiB
	rotateKeepBytes = 1 << 20 // 1 MiB
)
