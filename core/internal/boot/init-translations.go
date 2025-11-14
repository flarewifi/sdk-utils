//go:build !dev

package boot

import "core/internal/api"

func InitTranslationScan(g *api.CoreGlobals) {
	// No-op in production builds
}
