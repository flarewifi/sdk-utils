package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	sdkapi "sdk/api"

	"core/db/models"
	"core/internal/connmgr"
	webutil "core/internal/utils/web"
	"core/internal/web/helpers"
	"core/internal/web/middlewares"

	"github.com/gorilla/mux"
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
			acct, err := self.api.CoreAPI.HttpAPI.auth.IsAuthenticated(r)
			if err != nil {
				loginRoute := webutil.RootRouter.Get("admin:login")
				loginUrl, _ := loginRoute.URL()
				http.Redirect(w, r, loginUrl.String(), http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), sdkapi.SysAcctCtxKey, acct)
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

			tx, err := self.api.db.BeginTx(ctx, nil)
			if err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}
			defer tx.Rollback()

			client, err := helpers.CurrentClient(r)
			if err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}

			mdls := self.api.models
			device, err := mdls.Device().Find(tx, ctx, client.Id())
			if err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}

			purchase, err := mdls.Purchase().PendingPurchase(tx, ctx, device.Id())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				self.ErrorPage(w, err, errCode)
				return
			}

			if purchase != nil {
				self.api.HttpAPI.Response().Redirect(w, r, "payments:options")
				return
			}

			if err := tx.Commit(); err != nil {
				self.ErrorPage(w, err, errCode)
				return
			}

			next.ServeHTTP(w, r)

		})

		deviceMw := self.Device()
		return deviceMw(handler)
	}

}

func (self *PluginMiddlewares) TrackNav() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only track GET requests to avoid tracking form submissions
			if r.Method == http.MethodGet {
				// Get the current route
				route := mux.CurrentRoute(r)
				if route != nil {
					routeName := route.GetName()
					// Only track admin routes (check if route contains admin patterns)
					// Route names are formatted as: "com.domain.plugin#route_name"

					// Split by the # separator
					parts := strings.SplitN(routeName, "#", 2)

					// Filter only admin routes
					if len(parts) == 2 && strings.HasPrefix(parts[1], RouteNameAdminPrefix) {
						// Skip SSE routes and HTMX requests (which are typically for dynamic content loading)
						if !strings.Contains(routeName, RouteNameAdminSSE) && r.Header.Get("HX-Request") != "true" {
							// Track this navigation visit asynchronously
							params := mux.Vars(r)
							go self.trackNavVisit(parts[0], parts[1], params)
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (self *PluginMiddlewares) trackNavVisit(pluginPkg, pluginRouteName string, params map[string]string) {
	// Extract plugin package and actual route name
	// Route names are formatted as: "com.domain.plugin#route_name"
	// Using "#" as separator makes parsing clear and unambiguous

	routeParamsJSON := ""
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			fmt.Printf("Failed to marshal route params for quick access nav visit: %v\n", err)
		}
		routeParamsJSON = string(p)
	}

	// Track the visit
	ctx := context.Background()
	err := self.models.QuickAccessNav().Upsert(
		ctx,
		pluginPkg,
		pluginRouteName,
		routeParamsJSON,
	)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to track quick access nav visit: %v\n", err)
	}
}

func (self *PluginMiddlewares) ErrorPage(w http.ResponseWriter, err error, code int) {
	// TODO: Display common error page
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}
