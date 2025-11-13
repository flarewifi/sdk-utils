package api

import (
	"encoding/json"
	"net/http"
	sdkapi "sdk/api"

	"core/resources/views"
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
	_, themeApi, err := self.api.PluginsMgrApi.GetAdminTheme()
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

	_, themesAPI, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
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
		self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_type")
		self.api.HttpAPI.httpCookie.DeleteCookie(w, "flash_message")
	}

	sseURL := coreAPI.HttpAPI.Helpers().UrlForRoute("portal.sse")
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
		"hx-sse": "connect:" + sseURL,
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
	self.api.HttpAPI.Cookie().SetCookie(w, "flash_type", t)
	self.api.HttpAPI.Cookie().SetCookie(w, "flash_message", msg)
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
	http.Redirect(w, r, "/?t="+sdkutils.RandomStr(16), http.StatusSeeOther)
}

func (self *HttpResponse) Error(w http.ResponseWriter, r *http.Request, err error, status int) {
	// w.WriteHeader(status)
	page := views.ErrorPage(err)
	page.Render(r.Context(), w)
	// v := sdkhttp.ViewPage{PageContent: page}
	// _, autherr := self.api.HttpAPI.auth.CurrentAcct(r)
	// if autherr != nil {
	// 	self.api.HttpAPI.HttpResponse().PortalView(w, r, v)
	// } else {
	// 	self.api.HttpAPI.HttpResponse().AdminView(w, r, v)
	// }
}
