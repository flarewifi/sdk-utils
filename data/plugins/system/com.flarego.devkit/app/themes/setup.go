package themes

import (
	sdkapi "sdk/api"

	"com.flarego.devkit/app/routes"
	"com.flarego.devkit/app/web/navs"
)

// Setup registers the Devkit admin + portal themes and their routes, plus the
// developer plugin panel (upload / install / download plugin sources).
//
// com.flarego.devkit is a committed system plugin (data/plugins/system),
// statically linked into every local-dev and devkit build. It is intentionally
// active in ALL builds (no build-tag gate): the software-release builder strips
// it from production product images server-side, so it never reaches a device.
// Keeping it ungated lets developers iterate on the theme straight from a plain
// `make` dev run without building the devkit. The developer panel rides along on
// that same guarantee — its install action is additionally gated by the core's
// IsDevkit() check, so uploads only take effect inside a devkit build.
func Setup(api sdkapi.IPluginApi) error {
	SetAdminTheme(api)
	SetPortalTheme(api)
	SetupRoutes(api)

	// Developer panel: upload/install/download plugins under development.
	routes.AdminRoutes(api)
	navs.SetAdminNavs(api)
	return nil
}
