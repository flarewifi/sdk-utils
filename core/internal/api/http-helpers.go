package api

import (
	"html/template"
	"net/http"
	"path"

	sdkapi "sdk/api"

	"github.com/gorilla/csrf"
)

func NewHttpHelpers(api *PluginApi) sdkapi.IHttpHelpers {
	return &HttpHelpers{api: api}
}

type HttpHelpers struct {
	api *PluginApi
}

func (self *HttpHelpers) CsrfHtmlTag(r *http.Request) string {
	tpl := csrf.TemplateField(r)
	return string(tpl)
}

func (self *HttpHelpers) Translate(msgtype string, msgk string, pairs ...any) string {
	return self.api.Utl.Translate(msgtype, msgk, pairs...)
}

func (self *HttpHelpers) AdminAssetPath(p string) string {
	assets := self.api.AssetsManifest.AdminAssets
	if f, ok := assets.Scripts[p]; ok {
		return self.DistPath(f)
	}

	if f, ok := assets.Styles[p]; ok {
		return self.DistPath(f)
	}

	return ""
}

func (self *HttpHelpers) PortalAssetPath(p string) string {
	assets := self.api.AssetsManifest.PortalAssets
	if f, ok := assets.Scripts[p]; ok {
		return self.DistPath(f)
	}

	if f, ok := assets.Styles[p]; ok {
		return self.DistPath(f)
	}

	return ""
}

func (self *HttpHelpers) DistPath(p string) string {
	return path.Join("/assets/plugin", self.api.info.Package, self.api.info.Version, "resources/assets/dist", p)
}

func (self *HttpHelpers) PublicPath(p string) string {
	return path.Join("/assets/plugin", self.api.info.Package, self.api.info.Version, "resources/assets/public", p)
}

func (self *HttpHelpers) AdsView() (html template.HTML) {
	return ""
}

func (self *HttpHelpers) UrlForRoute(name string, pairs ...string) string {
	return self.api.HttpAPI.httpRouter.UrlForRoute(sdkapi.PluginRouteName(name), pairs...)
}

func (self *HttpHelpers) UrlForPkgRoute(pkg string, name string, pairs ...string) string {
	return self.api.HttpAPI.httpRouter.UrlForPkgRoute(pkg, name, pairs...)
}
