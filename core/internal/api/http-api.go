package api

import (
	"net/http"

	"core/db"
	"core/db/models"
	"core/internal/connmgr"
	"core/internal/web/helpers"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpApi(api *PluginApi, db *db.Database, clnt *connmgr.ClientRegister, mdls *models.Models, dmgr *connmgr.ClientRegister, pmgr *PaymentsMgr) {
	navs := NewNavsApi(api)
	auth := NewHttpAuth(api)
	httpResp := NewHttpResponse(api)
	middlewares := NewPluginMiddlewares(api, mdls, dmgr, pmgr)
	httpRouter := NewHttpRouterApi(api, db, clnt)
	httpForm := NewHttpFormApi(api)

	httpApi := &HttpApi{
		api:         api,
		auth:        auth,
		httpRouter:  httpRouter,
		navsApi:     navs,
		httpResp:    httpResp,
		middlewares: middlewares,
		formsApi:    httpForm,
	}

	api.HttpAPI = httpApi
}

type HttpApi struct {
	api         *PluginApi
	auth        *HttpAuth
	httpRouter  *HttpRouterApi
	navsApi     *HttpNavsApi
	formsApi    *HttpFormApi
	httpResp    *HttpResponse
	httpCookie  *HttpCookie
	middlewares *PluginMiddlewares
}

func (self *HttpApi) Initialize() {
	self.httpRouter.Initialize()
}

func (self *HttpApi) GetClientDevice(r *http.Request) (sdkapi.IClientDevice, error) {
	return helpers.CurrentClient(self.api.ClntReg, r)
}

func (self *HttpApi) Cookie() sdkapi.IHttpCookie {
	return self.httpCookie
}

func (self *HttpApi) Auth() sdkapi.IHttpAuth {
	return self.auth
}

func (self *HttpApi) Router() sdkapi.IHttpRouterApi {
	return self.httpRouter
}

func (self *HttpApi) Forms() sdkapi.IHttpFormsApi {
	return self.formsApi
}

func (self *HttpApi) Helpers() sdkapi.IHttpHelpers {
	return NewHttpHelpers(self.api)
}

func (self *HttpApi) Middlewares() sdkapi.IHttpMiddlewares {
	return self.middlewares
}

func (self *HttpApi) Response() sdkapi.IHttpResponse {
	return self.httpResp
}

func (self *HttpApi) MuxVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}

func (self *HttpApi) Navs() sdkapi.INavpsApi {
	return self.navsApi
}
