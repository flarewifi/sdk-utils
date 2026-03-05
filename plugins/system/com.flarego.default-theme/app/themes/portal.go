package themes

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/auth"
	"com.flarego.default-theme/resources/views/portal"
)

func SetPortalTheme(api sdkapi.IPluginApi) {

	api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		PreviewMeta: &sdkapi.ThemePreviewMeta{
			Background:     "#0d0d1a",
			CardColor:      "rgba(255, 255, 255, 0.15)",
			PrimaryColor:   "#b535ff",
			SecondaryColor: "#00bfff",
			AccentColor:    "#2F2EC7",
			ButtonColor:    "#000000",
			TextColor:      "#ffffff",
			LogoPosition:   "top",
		},
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			data := portal.PortalLayoutData{Components: c}
			layout := portal.PortalLayout(data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
			csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
			page := auth.LoginPage(api, csrfHtml, data)
			return sdkapi.ViewPage{
				Assets: sdkapi.ViewAssets{
					JsFile:  "auth/login.js",
					CssFile: "auth/login.css",
				},
				PageContent: page,
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			ctx := r.Context()
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				api.Logger().Error("Error in getting client device: " + err.Error())
				return sdkapi.ViewPage{}
			}

			summary, err := api.SessionsMgr().SessionSummary(ctx, clnt)
			if err != nil {
				api.Logger().Error("Error in session summary query: " + err.Error())
				api.Logger().Error("Error in getting client device: " + err.Error())
				return sdkapi.ViewPage{}
			}

			var sessionType sdkapi.SessionType
			runningSession, ok := api.SessionsMgr().RunningSession(clnt)
			if ok {
				sessionType = runningSession.Type()
			}
			navs := api.Http().Navs().GetPortalItems(r)
			page := portal.PortalIndexPage(api, portal.PortalIndexData{
				Navs:             navs,
				SessionSummary:   summary,
				IsSessionRunning: ok,
				SessionType:      sessionType,
				DeviceMac:        clnt.MacAddr(),
				DeviceIP:         clnt.IpAddr(),
			})

			return sdkapi.ViewPage{
				Assets: sdkapi.ViewAssets{
					JsFile: "portal/index.js",
				},
				PageContent: page,
			}
		},
	})
}
