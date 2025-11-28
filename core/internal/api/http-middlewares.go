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
	"core/internal/web/middlewares"
	"tools/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// WebhookClaims represents JWT claims for internal webhook authentication
type WebhookClaims struct {
	DeviceID    int64  `json:"device_id"`
	PurchaseUID string `json:"purchase_uid"`
	jwt.RegisteredClaims
}

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

func (self *PluginMiddlewares) CacheResponse(days int) func(http.Handler) http.Handler {
	return middlewares.CacheResponse(days)
}

func (self *PluginMiddlewares) HTTPSRedirect() func(http.Handler) http.Handler {
	return middlewares.HTTPSRedirect()
}

func (self *PluginMiddlewares) PendingPurchase() func(http.Handler) http.Handler {
	coreAPI := self.api.CoreAPI

	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			res := coreAPI.HttpAPI.Response()

			client, err := self.api.Http().GetClientDevice(r)
			if err != nil {
				res.FlashMsg(w, r, coreAPI.Translate("error", "Client device not registered"), sdkapi.FlashMsgError)
				res.RedirectToPortal(w, r)
				return
			}

			mdls := self.api.models
			purchase, err := mdls.Purchase().PendingPurchase(ctx, client.Id())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				res.FlashMsg(w, r, coreAPI.Translate("error", "Client device not registered"), sdkapi.FlashMsgError)
				res.RedirectToPortal(w, r)
				return
			}

			if purchase != nil {
				// If purchase is processing and has a payment URL, redirect to it
				if purchase.Processing() && purchase.PaymentUrl() != "" {
					http.Redirect(w, r, purchase.PaymentUrl(), http.StatusSeeOther)
					return
				}

				// Otherwise, redirect to payment options page with info message
				res.FlashMsg(w, r, coreAPI.Translate("info", "You have a pending purchase. Please complete it before proceeding."), sdkapi.FlashMsgInfo)
				res.Redirect(w, r, "payments:options")
				return
			}

			next.ServeHTTP(w, r)
		})

		return handler
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
		models.UpsertQuickAccessNavParams{
			PluginPkg:   pluginPkg,
			RouteName:   pluginRouteName,
			RouteParams: routeParamsJSON,
		},
	)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to track quick access nav visit: %v\n", err)
	}
}

// WebhookAuth verifies JWT token for internal webhook requests
func (self *PluginMiddlewares) WebhookAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Missing Authorization header"}`))
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid Authorization header format"}`))
				return
			}

			tokenString := parts[1]

			// Get application secret
			appCfg, err := config.ReadApplicationConfig()
			if err != nil {
				fmt.Println("WebhookAuth: Failed to read application config:", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Server configuration error"}`))
				return
			}

			// Parse and verify token
			token, err := jwt.ParseWithClaims(tokenString, &WebhookClaims{}, func(token *jwt.Token) (any, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(appCfg.Secret), nil
			})

			if err != nil {
				fmt.Println("WebhookAuth: Token verification failed:", err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid token"}`))
				return
			}

			// Extract claims
			if claims, ok := token.Claims.(*WebhookClaims); ok && token.Valid {
				fmt.Printf("WebhookAuth: Valid token for device %d, purchase %s\n", claims.DeviceID, claims.PurchaseUID)

				// Add claims to context
				ctx := context.WithValue(r.Context(), sdkapi.WebhookDeviceIDKey, claims.DeviceID)
				ctx = context.WithValue(ctx, sdkapi.WebhookPurchaseUIDKey, claims.PurchaseUID)

				// Continue with the authenticated request
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				fmt.Println("WebhookAuth: Invalid token claims")
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid token claims"}`))
				return
			}
		})
	}
}

func (self *PluginMiddlewares) ErrorPage(w http.ResponseWriter, err error, code int) {
	// TODO: Display common error page
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}
