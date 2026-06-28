package themes

import sdkapi "sdk/api"

// Setup registers the Devkit admin + portal themes and their routes.
//
// com.flarego.devkit is a committed system plugin (data/plugins/system),
// statically linked into every local-dev and devkit build. It is intentionally
// active in ALL builds (no build-tag gate): the software-release builder strips
// it from production product images server-side, so it never reaches a device.
// Keeping it ungated lets developers iterate on the theme straight from a plain
// `make` dev run without building the devkit.
func Setup(api sdkapi.IPluginApi) error {
	SetAdminTheme(api)
	SetPortalTheme(api)
	SetupRoutes(api)
	return nil
}
