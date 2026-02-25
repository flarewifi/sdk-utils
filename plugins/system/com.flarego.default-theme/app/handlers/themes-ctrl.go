package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

type themesConfig struct {
	PortalThemePkg string `json:"portal"`
	AdminThemePkg  string `json:"admin"`
}

func readCurrentThemesConfig() themesConfig {
	cfg := themesConfig{
		PortalThemePkg: "com.flarego.default-theme",
		AdminThemePkg:  "com.flarego.default-theme",
	}
	configFile := filepath.Join(sdkutils.PathConfigDir, "themes.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	if cfg.PortalThemePkg == "" {
		cfg.PortalThemePkg = "com.flarego.default-theme"
	}
	if cfg.AdminThemePkg == "" {
		cfg.AdminThemePkg = "com.flarego.default-theme"
	}
	return cfg
}

func buildThemeCards(api sdkapi.IPluginApi, feature string, currentPkg string) []admin.ThemeCard {
	var cards []admin.ThemeCard
	allPlugins := api.PluginsMgr().All()
	for _, p := range allPlugins {
		features := p.Features()
		for _, f := range features {
			if f == feature {
				info := p.Info()
				cards = append(cards, admin.ThemeCard{
					Package:     info.Package,
					Name:        info.Name,
					Description: info.Description,
					IsCurrent:   info.Package == currentPkg,
				})
				break
			}
		}
	}
	return cards
}

func AdminThemesPageCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := readCurrentThemesConfig()
		cards := buildThemeCards(api, "theme:admin", cfg.AdminThemePkg)
		saveUrl := api.Http().Helpers().UrlForPkgRoute("com.flarego.core", "admin:themes:save")

		page := admin.AdminThemesPage(api, cards, cfg.AdminThemePkg, saveUrl, cfg.PortalThemePkg)
		api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				CssFile: "admin/themes.css",
				JsFile:  "admin/themes.js",
			},
			PageContent: page,
		})
	}
}

func PortalThemesPageCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := readCurrentThemesConfig()
		cards := buildThemeCards(api, "theme:portal", cfg.PortalThemePkg)
		saveUrl := api.Http().Helpers().UrlForPkgRoute("com.flarego.core", "admin:themes:save")

		page := admin.PortalThemesPage(api, cards, cfg.PortalThemePkg, saveUrl, cfg.AdminThemePkg)
		api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				CssFile: "admin/themes.css",
				JsFile:  "admin/themes.js",
			},
			PageContent: page,
		})
	}
}
