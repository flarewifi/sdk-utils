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
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			data := portal.PortalLayoutData{Components: c}
			layout := portal.PortalLayout(data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
			csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
			page := auth.LoginPage(csrfHtml, data)
			return sdkapi.ViewPage{PageContent: page}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				api.Logger().Error("Error in getting client device: " + err.Error())
				return sdkapi.ViewPage{}
			}

			ctx := r.Context()
			tx, err := api.SqlDb().Begin(ctx)
			if err != nil {
				api.Logger().Error("Error initializing transaction: " + err.Error())
				return sdkapi.ViewPage{}
			}
			defer tx.Rollback(ctx)

			summary, err := api.SessionsMgr().SessionSummary(tx, ctx, clnt)
			if err != nil {
				api.Logger().Error("Error in session summary query: " + err.Error())
				return sdkapi.ViewPage{}
			}

			_, ok := api.SessionsMgr().CurrSession(clnt)
			navs := api.Http().Navs().GetPortalItems(r)
			page := portal.PortalIndexPage(api, portal.PortalIndexData{
				Navs:             navs,
				SessionSummary:   summary,
				IsSessionRunning: ok,
			})

			if err := tx.Commit(ctx); err != nil {
				api.Logger().Error("Error committing db transaction: " + err.Error())
				return sdkapi.ViewPage{}
			}

			return sdkapi.ViewPage{
				Assets: sdkapi.ViewAssets{
					JsFile: "portal/index.js",
				},
				PageContent: page,
			}
		},
	})
}
