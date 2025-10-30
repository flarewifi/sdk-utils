package boot

import (
	"core/internal/api"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitAssets(g *api.CoreGlobals) {
	// Compute all the global assets hash
	var globalAdminJs, globalAdminCss []string
	var globalPortalJs, globalPortalcss []string

	for _, p := range g.PluginMgr.All() {
		api := p.(*api.PluginApi)
		manifest := api.AssetsManifest
		globalJsFiles := []string{"global.js", "globals.js"}
		globalCssFiles := []string{"global.css", "globals.css"}

		for _, jsfname := range globalJsFiles {
			globalJs, ok := manifest.AdminAssets.Scripts[jsfname]
			file := filepath.Join(p.Resource("assets/dist/" + globalJs))
			if ok && sdkutils.FsExists(file) {
				globalAdminJs = append(globalAdminJs, file)
			}

			globalJs, ok = manifest.PortalAssets.Scripts[jsfname]
			file = filepath.Join(p.Resource("assets/dist/" + globalJs))
			if ok && sdkutils.FsExists(file) {
				globalPortalJs = append(globalPortalJs, file)
			}
		}

		for _, cssfname := range globalCssFiles {
			globalCss, ok := manifest.AdminAssets.Styles[cssfname]
			file := filepath.Join(p.Resource("assets/dist/" + globalCss))
			if ok && sdkutils.FsExists(file) {
				globalAdminCss = append(globalAdminCss, file)
			}

			globalCss, ok = manifest.PortalAssets.Styles[cssfname]
			file = filepath.Join(p.Resource("assets/dist/" + globalCss))
			if ok && sdkutils.FsExists(file) {
				globalPortalcss = append(globalPortalcss, filepath.Join(p.Resource("assets/dist/"+globalCss)))
			}
		}
	}

	adminJsHash := sdkutils.Sha1Hash(globalAdminJs...)
	adminCssHash := sdkutils.Sha1Hash(globalAdminCss...)
	portalJsHash := sdkutils.Sha1Hash(globalPortalJs...)
	portalCssHash := sdkutils.Sha1Hash(globalPortalcss...)

	g.GlobalAssets.AdminJsHash = adminJsHash
	g.GlobalAssets.AdminJsFiles = globalAdminJs
	g.GlobalAssets.AdminCssHash = adminCssHash
	g.GlobalAssets.AdminCssFiles = globalAdminCss
	g.GlobalAssets.PortalJsHash = portalJsHash
	g.GlobalAssets.PortalJsFiles = globalPortalJs
	g.GlobalAssets.PortalCssHash = portalCssHash
	g.GlobalAssets.PortalCssFiles = globalPortalcss
}
