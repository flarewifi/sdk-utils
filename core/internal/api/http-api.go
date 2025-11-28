package api

import (
	"fmt"
	"net/http"

	"core/db"
	"core/db/models"
	"core/internal/connmgr"
	devicetoken "core/internal/utils/device-token"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpApi(api *PluginApi, db *db.Database, assets *GlobalAssets, clnt *connmgr.ClientRegister, mdls *models.Models, dmgr *connmgr.ClientRegister, pmgr *PaymentsMgr) {
	navs := NewNavsApi(api)
	auth := NewHttpAuth(api)
	httpResp := NewHttpResponse(api, assets)
	middlewares := NewPluginMiddlewares(api, mdls, dmgr, pmgr)
	httpRouter := NewHttpRouterApi(api, db, clnt)
	httpForm := NewHttpFormApi(api)
	httpCookie := NewHttpCookie(api)

	httpApi := &HttpApi{
		api:            api,
		auth:           auth,
		httpRouter:     httpRouter,
		navsApi:        navs,
		httpResp:       httpResp,
		middlewares:    middlewares,
		formsApi:       httpForm,
		httpCookie:     httpCookie,
		clientRegister: clnt,
	}

	api.HttpAPI = httpApi
}

type HttpApi struct {
	api            *PluginApi
	auth           *HttpAuth
	httpRouter     *HttpRouterApi
	navsApi        *HttpNavsApi
	formsApi       *HttpFormApi
	httpResp       *HttpResponse
	httpCookie     *HttpCookie
	middlewares    *PluginMiddlewares
	clientRegister *connmgr.ClientRegister
}

func (self *HttpApi) Initialize() {
	self.httpRouter.Initialize()
}

func (self *HttpApi) GetClientDevice(r *http.Request) (sdkapi.IClientDevice, error) {
	var deviceID int64
	var err error

	// Priority 1: Check X-Device-Token header (localStorage-based AJAX auth)
	deviceID, err = devicetoken.GetDeviceFromHeader(r)
	if err != nil {
		// Priority 2: Fallback to cookie (traditional auth)
		deviceID, err = devicetoken.GetDeviceCookie(r)
		if err != nil {
			return nil, fmt.Errorf("failed to get device from header or cookie: %w", err)
		}
	}

	// Query device from database
	ctx := r.Context()
	clnt, err := self.clientRegister.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return clnt, nil
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
