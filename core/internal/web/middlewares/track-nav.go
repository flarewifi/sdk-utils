package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"core/db/models"

	"github.com/gorilla/mux"
)

const (
	RouteNameAdminPrefix = "admin:"
	RouteNameAdminSSE    = "admin:sse"
)

// TrackNav tracks navigation visits for the Quick Access menu.
func TrackNav(mdls *models.Models) func(http.Handler) http.Handler {
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
							go trackNavVisit(mdls, parts[0], parts[1], params)
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func trackNavVisit(mdls *models.Models, pluginPkg, pluginRouteName string, params map[string]string) {
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
	err := mdls.QuickAccessNav().Upsert(
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
