package plugins

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	sdkhttp "sdk/api/http"

	"core/internal/connmgr"
	"core/internal/db/models"
	webutil "core/internal/utils/web"
	"core/internal/web/helpers"
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acct, err := webutil.IsAdminAuthenticated(r)
			if err != nil {
				loginRoute := webutil.RootRouter.Get("admin:login")
				loginUrl, _ := loginRoute.URL()
				http.Redirect(w, r, loginUrl.String(), http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), sdkhttp.SysAcctCtxKey, acct)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (self *PluginMiddlewares) Device() func(http.Handler) http.Handler {
	return middlewares.DeviceMiddleware(self.api.db, self.creg)
}

func (self *PluginMiddlewares) CacheResponse(days int) func(http.Handler) http.Handler {
	return middlewares.CacheResponse(days)
}

func (self *PluginMiddlewares) PendingPurchase() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			errCode := http.StatusInternalServerError

			client, err := helpers.CurrentClient(self.api.ClntReg, r)
			if err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}

			mdls := self.api.models
			device, err := mdls.Device().Find(ctx, client.Id())
			if err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}

			purchase, err := mdls.Purchase().PendingPurchase(ctx, device.Id())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				self.ErrorPage(w, err, errCode)
				return
			}

			if purchase != nil {
				self.api.HttpAPI.HttpResponse().Redirect(w, r, "payments:customer:options")
				return
			}

			next.ServeHTTP(w, r)

		})

		deviceMw := self.Device()
		return deviceMw(handler)
	}

}

func (self *PluginMiddlewares) ErrorPage(w http.ResponseWriter, err error, code int) {
	// TODO: Display common error page
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}
