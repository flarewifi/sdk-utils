package middlewares

import (
	"context"
	"net/http"

	webutil "core/internal/utils/web"
	sdkapi "sdk/api"
)

// AdminAuth authenticates the user as admin.
// Redirects to login page if not authenticated.
func AdminAuth(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acct, err := api.Http().Auth().IsAuthenticated(r)
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
