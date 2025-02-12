package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"core/resources/views"
	"core/resources/views/themes"

	"github.com/a-h/templ"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

type HttpResponse struct {
	api      *PluginApi
	viewroot string
}

func NewHttpResponse(api *PluginApi) *HttpResponse {
	viewroot := sdkutils.StripRootPath(api.Utl.Resource("views"))
	return &HttpResponse{api, viewroot}
}

func (self *HttpResponse) AdminView(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	_, themeApi, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	sseURL := self.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:sse")
	navs := self.api.HttpAPI.navsApi.GetAdminNavs(r)
	assets := self.api.Utl.GetAdminAssetsForPage(v)

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

	layoutBuilder := &ThemesLayoutBuilder{
		PageContent: v.PageContent,
		ContentWrapper: func(head, layout templ.Component) {
			data := themes.AdminLayoutData{
				Assets: assets,
				SseURL: sseURL,
				Flash:  flash,
				Head:   head,
				Layout: layout,
			}

			page := themes.AdminThemeLayout(data)
			if err := page.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>TemplateError: %s</p>", err.Error())
			}
		},
	}

	data := sdkapi.AdminThemeData{
		Builder: layoutBuilder,
		Navs:    navs,
	}

	w.Header().Set("Content-Type", "text/html")
	themeApi.AdminTheme.LayoutFactory(w, r, data)
}

func (self *HttpResponse) PortalView(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	api := self.api.CoreAPI
	q := r.URL.Query()
	pageUUID := q.Get("t") // Prevent caching

	_, themeApi, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	sseURL := api.HttpAPI.Helpers().UrlForRoute("portal:sse")
	ssePolyfillURL := api.Http().Helpers().PortalAssetPath("polyfills.js")
	assets := api.Utl.GetPortalAssetsForPage(v)

	fmt.Println("SSE Polyfill URL: ", ssePolyfillURL)
	fmt.Printf("Manifest: %+v\n", self.api.AssetsManifest.PortalAssets)

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

	layoutBuilder := &ThemesLayoutBuilder{
		PageContent: v.PageContent,
		ContentWrapper: func(head, layout templ.Component) {
			data := themes.PortalLayoutData{
				PageUUID:       pageUUID,
				Assets:         assets,
				SseURL:         sseURL,
				SsePolyfillURL: ssePolyfillURL,
				Flash:          flash,
				Head:           head,
				Layout:         layout,
			}

			page := themes.PortalThemeLayout(data)
			if err := page.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>TemplateError: %s</p>", err.Error())
			}
		},
	}

	data := sdkapi.PortalThemeData{
		Builder: layoutBuilder,
	}

	w.Header().Set("Content-Type", "text/html")
	themeApi.PortalTheme.LayoutFactory(w, r, data)
}

func (self *HttpResponse) View(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	w.Header().Set("Content-Type", "text/html")
	if err := v.PageContent.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

func (self *HttpResponse) Json(w http.ResponseWriter, r *http.Request, data interface{}, status int) {
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
	http.Redirect(w, r, url, http.StatusSeeOther)
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
