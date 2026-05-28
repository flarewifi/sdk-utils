package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	sdkapi "sdk/api"

	adminview "core/resources/views/admin/error"
	portalview "core/resources/views/portal"
	"core/resources/views/themes"

	"github.com/a-h/templ"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

type HttpResponse struct {
	api      *PluginApi
	assets   *GlobalAssets
	viewroot string
}

func NewHttpResponse(api *PluginApi, assets *GlobalAssets) *HttpResponse {
	viewroot := sdkutils.StripRootPath(api.Utl.Resource("views"))
	return &HttpResponse{api, assets, viewroot}
}

func (self *HttpResponse) AdminView(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	_, themeApi, isFallback, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	sseURL := self.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:sse")
	assets, err := GetAdminAssetsForPage(self.api.CoreAPI, themeApi.api, self.api, v, self.assets)
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	var flash *sdkapi.FlashMsg
	if isFallback {
		msg := self.api.Translate("warning", "The selected theme is not valid or not installed. Please select a valid theme.")
		flash = &sdkapi.FlashMsg{
			Type:    sdkapi.FlashMsgWarning,
			Message: msg,
		}
	} else {
		flashType, _ := self.api.HttpAPI.httpCookie.GetCookie(r, "flash_type")
		flashMsg, _ := self.api.HttpAPI.httpCookie.GetCookie(r, "flash_message")
		if flashType != "" && flashMsg != "" {
			flash = &sdkapi.FlashMsg{
				Type:    flashType,
				Message: flashMsg,
			}
			self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_type")
			self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_message")
		}
	}

	htmlAttrs := templ.Attributes{}
	bodyAttrs := templ.Attributes{
		"hx-ext":      "sse,loading-states",
		"sse-connect": sseURL,
	}

	head := themes.AdminHead(self.api, assets)
	scripts := themes.AdminScripts(assets, flash)

	layoutBuilder := &ThemesLayoutBuilder{
		htmlAttrs:      htmlAttrs,
		headContent:    head,
		bodyAttrs:      bodyAttrs,
		pageContent:    v.PageContent,
		scriptsContent: scripts,
	}

	w.Header().Set("Content-Type", "text/html")
	themeApi.AdminTheme.LayoutBuilder(w, r, layoutBuilder)
}

func (self *HttpResponse) PortalView(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	q := r.URL.Query()
	pageUUID := q.Get("t") // Prevent caching
	coreAPI := self.api.CoreAPI

	_, themesAPI, isFallback, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	if isFallback {
		msg := self.api.Translate("warning", "The selected theme is not valid or not installed. Please select a valid theme.")
		self.api.HttpAPI.httpCookie.SetCookie(w, "flash_type", sdkapi.FlashMsgWarning, nil)
		self.api.HttpAPI.httpCookie.SetCookie(w, "flash_message", msg, nil)
	}

	assets, err := GetPortalAssetsForPage(coreAPI, themesAPI.api, self.api, v, self.assets)
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	var flash *sdkapi.FlashMsg
	flashType, _ := self.api.HttpAPI.httpCookie.GetCookie(r, "flash_type")
	flashMsg, _ := self.api.HttpAPI.httpCookie.GetCookie(r, "flash_message")
	if flashType != "" && flashMsg != "" {
		flash = &sdkapi.FlashMsg{
			Type:    flashType,
			Message: flashMsg,
		}
		// Only delete flash cookies if PreserveFlash is false
		if !v.PreserveFlash {
			self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_type")
			self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_message")
		}
	}

	sseURL := coreAPI.HttpAPI.Helpers().UrlForRoute("portal:sse")
	polyfillsURL := coreAPI.Http().Helpers().PortalAssetPath("polyfills.js")
	data := themes.PortalLayoutData{
		PageUUID:     pageUUID,
		Assets:       assets,
		SseURL:       sseURL,
		PolyfillsURL: polyfillsURL,
		Flash:        flash,
	}
	head := themes.PortalHead(self.api.CoreAPI, data)
	scripts := themes.PortalScripts(data.Assets, flash)
	htmlAttrs := templ.Attributes{}
	bodyAttrs := templ.Attributes{
		"hx-ext":      "sse",
		"sse-connect": sseURL,
	}

	layoutBuilder := &ThemesLayoutBuilder{
		htmlAttrs:      htmlAttrs,
		headContent:    head,
		bodyAttrs:      bodyAttrs,
		pageContent:    v.PageContent,
		scriptsContent: scripts,
	}

	w.Header().Set("Content-Type", "text/html")
	themesAPI.PortalTheme.LayoutBuilder(w, r, layoutBuilder)
}

func (self *HttpResponse) View(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	w.Header().Set("Content-Type", "text/html")
	if err := v.PageContent.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

func (self *HttpResponse) Json(w http.ResponseWriter, r *http.Request, data any, status int) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (self *HttpResponse) FlashMsg(w http.ResponseWriter, r *http.Request, msg string, t string) {
	self.api.HttpAPI.Cookie().SetCookie(w, "flash_type", t, nil)
	self.api.HttpAPI.Cookie().SetCookie(w, "flash_message", msg, nil)
}

func (self *HttpResponse) Redirect(w http.ResponseWriter, r *http.Request, routeName string, pairs ...string) {
	url := self.api.HttpAPI.Helpers().UrlForRoute(routeName, pairs...)
	if r.Header.Get("Hx-Request") == "true" {
		w.Header().Add("Hx-Redirect", url)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

func (self *HttpResponse) RedirectToPortal(w http.ResponseWriter, r *http.Request) {
	url := self.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")
	w.Header().Add("Hx-Redirect", url)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (self *HttpResponse) RedirectSuccess(w http.ResponseWriter, r *http.Request, redirectURL string) {
	page := portalview.PortalSuccessRedirectPage(self.api.CoreAPI, redirectURL)
	v := sdkapi.ViewPage{
		Assets: sdkapi.ViewAssets{
			JsFile:  "portal-success-redirect.js",
			CssFile: "portal-success-redirect.css",
		},
		PageContent: page,
	}
	self.api.CoreAPI.HttpAPI.Response().PortalView(w, r, v)
}

func (self *HttpResponse) Error(w http.ResponseWriter, r *http.Request, err error, status int) {
	// Add panic recovery for template rendering failures
	defer func() {
		if r := recover(); r != nil {
			self.api.LoggerAPI.Error(fmt.Sprintf("Error page rendering failed: %v", r))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "An error occurred. Please try again later.")
		}
	}()

	// Log internal error with full details
	self.api.LoggerAPI.Error(fmt.Sprintf("HTTP Error: %s - Status: %d - Path: %s", err.Error(), status, r.URL.Path))

	// Sanitize error message for users (SECURITY)
	userMsg, sanitizedStatus := SanitizeError(self.api, err)

	// Detect if admin or portal based on URL path
	isAdmin := strings.HasPrefix(r.URL.Path, "/admin")

	if isAdmin {
		// Admin error page (Bootstrap 5)
		// Use CoreAPI to ensure error page assets are loaded from core, not the calling plugin
		page := adminview.AdminErrorPage(self.api.CoreAPI, userMsg)
		v := sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				JsFile: "admin-error.js",
				// NO CSS FILE - admin uses only Bootstrap 5
			},
			PageContent: page,
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(sanitizedStatus)
		self.api.CoreAPI.HttpAPI.Response().AdminView(w, r, v)
	} else {
		// Portal error page (Bootstrap 3)
		// Use CoreAPI to ensure error page assets are loaded from core, not the calling plugin
		page := portalview.PortalErrorPage(self.api.CoreAPI, userMsg)
		v := sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				JsFile:  "portal-error.js",
				CssFile: "portal-error.css",
			},
			PageContent: page,
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(sanitizedStatus)
		self.api.CoreAPI.HttpAPI.Response().PortalView(w, r, v)
	}
}
