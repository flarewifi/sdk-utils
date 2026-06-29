package coretheme

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"

	corethemeauth "core/resources/views/themes/fallback/auth"
	corethemeportal "core/resources/views/themes/fallback/portal"
)

// SetPortalTheme registers the built-in core portal theme on CoreAPI.
func SetPortalTheme(api sdkapi.IPluginApi) {
	api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
		JsFile:  "theme-fallback.js",
		CssFile:      "theme-fallback.css",
		PreviewImage: "",
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			data := corethemeportal.PortalLayoutData{Components: c}
			layout := corethemeportal.PortalLayout(data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
			csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
			page := corethemeauth.LoginPage(api, csrfHtml, data)
			return sdkapi.ViewPage{
				Assets:      sdkapi.ViewAssets{},
				PageContent: page,
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			ctx := r.Context()
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				return sdkapi.ViewPage{}
			}

			summary, err := api.SessionsMgr().SessionSummary(ctx, clnt)
			if err != nil {
				return sdkapi.ViewPage{}
			}

			var sessionType sdkapi.SessionType
			runningSession, sessionRunning := api.SessionsMgr().RunningSession(clnt)
			if sessionRunning {
				sessionType = runningSession.Type()
			}

			navs := api.Http().Navs().GetPortalItems(r)
			page := corethemeportal.PortalIndexPage(api, corethemeportal.PortalIndexData{
				Navs:             navs,
				SessionSummary:   summary,
				IsSessionRunning: sessionRunning,
				SessionType:      sessionType,
				DeviceMac:        clnt.MacAddr(),
				DeviceIP:         clnt.IpAddr(),
			})

			return sdkapi.ViewPage{
				Assets:      sdkapi.ViewAssets{},
				PageContent: page,
			}
		},
	})
}
