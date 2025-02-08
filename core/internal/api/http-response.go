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

	layoutBuilder := &ThemesLayoutBuilder{
		FlashMessage: sdkapi.FlashMsg{},
		PageContent:  v.PageContent,
		ContentWrapper: func(head, layout templ.Component) {
			data := themes.AdminLayoutData{
				Assets: assets,
				SseURL: sseURL,
				Head:   head,
				Layout: layout,
			}

			page := themes.AdminThemeLayout(data)
			if err := page.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>TemplateError: %s</p>", err.Error())
			}
		},
	}

	data := sdkapi.AdminLayoutData{
		Api:     self.api,
		Builder: layoutBuilder,
		Navs:    navs,
	}

	w.Header().Set("Content-Type", "text/html")
	themeApi.AdminTheme.LayoutFactory(w, r, data)
}

func (self *HttpResponse) PortalView(w http.ResponseWriter, r *http.Request, v sdkapi.ViewPage) {
	_, themeApi, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		self.Error(w, r, err, http.StatusInternalServerError)
		return
	}

	sseURL := self.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:sse")
	assets := self.api.Utl.GetPortalAssetsForPage(v)

	layoutBuilder := &ThemesLayoutBuilder{
		FlashMessage: sdkapi.FlashMsg{},
		PageContent:  v.PageContent,
		ContentWrapper: func(head, layout templ.Component) {
			data := themes.PortalLayoutData{
				Assets: assets,
				SseURL: sseURL,
				Head:   head,
				Layout: layout,
			}

			page := themes.PortalThemeLayout(data)
			if err := page.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>TemplateError: %s</p>", err.Error())
			}
		},
	}

	data := sdkapi.PortalLayoutData{
		Api:     self.api,
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

}

func (self *HttpResponse) Redirect(w http.ResponseWriter, r *http.Request, routeName string, pairs ...string) {
	url := self.api.HttpAPI.Helpers().UrlForRoute(routeName, pairs...)
	w.Header().Set("HX-Redirect", url)
	http.Redirect(w, r, url, http.StatusSeeOther)
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
