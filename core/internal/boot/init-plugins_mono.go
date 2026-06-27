//go:build mono

package boot

import (
	"core/internal/api"
)

// loaderEmitsPluginProgress is false for mono: plugins are statically linked and
// registered in one shot by LoadMonoPlugins (no per-plugin load step to show), so
// the per-plugin booting-page checklist is emitted later, by InitLoadedPlugins as
// each plugin's Init runs — which is the visibly slow phase for mono.
const loaderEmitsPluginProgress = false

// InitPlugins returns an error for signature parity with the non-mono loader so
// the shared boot path (init.go) can handle both. Mono plugins are statically
// linked into the binary, so a compile failure is caught at build time rather
// than here; the generated LoadMonoPlugins cannot surface a load error, hence nil.
func InitPlugins(g *api.CoreGlobals) error {
	// Mono plugins are statically linked, so there is no compile phase and loading
	// is a single in-process registration. Advance the booting page to "Loading
	// plugins"; the per-plugin checklist is emitted later by InitLoadedPlugins as
	// each Init runs (see loaderEmitsPluginProgress).
	g.BootProgress.Advance(g.CoreAPI.Translate("info", "Loading plugins"))
	g.PluginMgr.LoadMonoPlugins(g)
	return nil
}
