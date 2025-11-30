package api

import (
	"net/http"

	"core/db/models"
	"core/internal/connmgr"
	"core/internal/web/middlewares"
)

func NewPluginMiddlewares(api *PluginApi, mdls *models.Models, dmgr *connmgr.ClientRegister, pmgr *PaymentsMgr) *PluginMiddlewares {
	return &PluginMiddlewares{api, mdls, dmgr, pmgr}
}

type PluginMiddlewares struct {
	api    *PluginApi
	models *models.Models
	creg   *connmgr.ClientRegister
	pmgr   *PaymentsMgr
}

func (self *PluginMiddlewares) AdminAuth() func(http.Handler) http.Handler {
	return middlewares.AdminAuth(self.api.CoreAPI)
}

func (self *PluginMiddlewares) CacheResponse(days int) func(http.Handler) http.Handler {
	return middlewares.CacheResponse(days)
}

func (self *PluginMiddlewares) HTTPSRedirect() func(http.Handler) http.Handler {
	return middlewares.HTTPSRedirect()
}

func (self *PluginMiddlewares) PendingPurchase() func(http.Handler) http.Handler {
	return middlewares.PendingPurchase(self.api.CoreAPI, self.models)
}

func (self *PluginMiddlewares) WebhookAuth() func(http.Handler) http.Handler {
	return middlewares.WebhookAuth()
}

func (self *PluginMiddlewares) ErrorPage(w http.ResponseWriter, err error, code int) {
	// TODO: Display common error page
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}
