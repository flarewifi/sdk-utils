package boot

import (
	"core/internal/api"
	"path/filepath"
	"slices"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func InitAssets(g *api.CoreGlobals) {
	// Compute all the global assets hash
	var globalAdminJs, globalAdminCss []string
	var globalPortalJs, globalPortalcss []string
	processedFiles := []string{}

	for _, p := range g.PluginMgr.Plugins() {
		api := p.(*api.PluginApi)
		manifest := api.AssetsManifest
		globalJsFiles := []string{"global.js", "globals.js"}
		globalCssFiles := []string{"global.css", "globals.css"}

		for _, jsfname := range globalJsFiles {
			globalJs, ok := manifest.AdminAssets.Scripts[jsfname]
			file := filepath.Join(p.Resource("assets/dist/" + globalJs))
			if ok && sdkutils.FsExists(file) && !slices.Contains(processedFiles, file) {
				globalAdminJs = append(globalAdminJs, file)
				processedFiles = append(processedFiles, file)
			}

			globalJs, ok = manifest.PortalAssets.Scripts[jsfname]
			file = filepath.Join(p.Resource("assets/dist/" + globalJs))
			if ok && sdkutils.FsExists(file) && !slices.Contains(processedFiles, file) {
				globalPortalJs = append(globalPortalJs, file)
				processedFiles = append(processedFiles, file)
			}
		}

		for _, cssfname := range globalCssFiles {
			globalCss, ok := manifest.AdminAssets.Styles[cssfname]
			file := filepath.Join(p.Resource("assets/dist/" + globalCss))
			if ok && sdkutils.FsExists(file) && !slices.Contains(processedFiles, file) {
				globalAdminCss = append(globalAdminCss, file)
				processedFiles = append(processedFiles, file)
			}

			globalCss, ok = manifest.PortalAssets.Styles[cssfname]
			file = filepath.Join(p.Resource("assets/dist/" + globalCss))
			if ok && sdkutils.FsExists(file) && !slices.Contains(processedFiles, file) {
				globalPortalcss = append(globalPortalcss, filepath.Join(p.Resource("assets/dist/"+globalCss)))
				processedFiles = append(processedFiles, file)
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
