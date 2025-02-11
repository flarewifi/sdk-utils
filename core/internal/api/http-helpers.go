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

func (self *HttpHelpers) Translate(msgtype string, msgk string, pairs ...interface{}) string {
	return self.api.Utl.Translate(msgtype, msgk, pairs...)
}

func (self *HttpHelpers) AssetPath(p string) string {
	// TODO: refer to the dist manifest file
	return path.Join("/plugin", self.api.info.Package, self.api.info.Version, "assets", "dist", p)
}

func (self *HttpHelpers) ResourcePath(p string) string {
	return path.Join("/plugin", self.api.info.Package, self.api.info.Version, "resources", p)
}

func (self *HttpHelpers) PluginMgr() sdkapi.IPluginsMgrApi {
	return self.api.PluginsMgrApi
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
