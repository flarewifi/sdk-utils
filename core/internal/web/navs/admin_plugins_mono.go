//go:build mono

package navs

import (
	"core/internal/api"
	sdkapi "sdk/api"
)

func GetAdminPluginNavs(g *api.CoreGlobals) []sdkapi.AdminNavItemOpt {
	// No plugin navs in monolithic build
	return []sdkapi.AdminNavItemOpt{}
}
