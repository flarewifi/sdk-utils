//go:build mono

package boot

import (
	"core/internal/api"
)

// InitPlugins returns an error for signature parity with the non-mono loader so
// the shared boot path (init.go) can handle both. Mono plugins are statically
// linked into the binary, so a compile failure is caught at build time rather
// than here; the generated LoadMonoPlugins cannot surface a load error, hence nil.
func InitPlugins(g *api.CoreGlobals) error {
	g.PluginMgr.LoadMonoPlugins(g)
	return nil
}
