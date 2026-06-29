package themes

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.devkit/resources/views/auth"
	"com.flarego.devkit/resources/views/portal"
)

// SetPortalTheme registers the Devkit portal theme: a basic Bootstrap 3 captive
// portal layout with a simple login and landing page. It intentionally avoids
// session/routing dependencies so it renders in the devkit (no real routing).
func SetPortalTheme(api sdkapi.IPluginApi) {
	api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
		JsFile:       "theme.js",
		CssFile:      "theme.css",
		CssLib:       sdkapi.CssLibBootstrap3,
		PreviewImage: "images/devkit-portal.svg",
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			data := portal.PortalLayoutData{Components: c}
			layout := portal.PortalLayout(data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
			return loginViewPage(api, r, data)
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			// Devkit portal index avoids session lookups (no real routing in the
			// devkit). It still surfaces any portal nav items registered by the
			// developer's plugin.
			navs := api.Http().Navs().GetPortalItems(r)
			return sdkapi.ViewPage{
				Assets:      sdkapi.ViewAssets{},
				PageContent: portal.PortalIndexPage(api, navs),
			}
		},
	})
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// loginViewPage builds the login ViewPage. It is shared by the portal theme's
// LoginPageFactory (initial render of /login) and the HTTPS-only GET /login
// handler (the re-render landing spot for a 302-downgraded login POST), so both
// paths produce an identical, CSRF-protected form.
func loginViewPage(api sdkapi.IPluginApi, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
	csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
	return sdkapi.ViewPage{
		Assets:      sdkapi.ViewAssets{},
		PageContent: auth.LoginPage(api, csrfHtml, data),
	}
}
