package middlewares

// import (
// 	"core/internal/config"
// 	"net/http"

// 	"github.com/gorilla/csrf"
// )

// func CsrfMiddleware() func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		var csrfMiddleware func(http.Handler) http.Handler

// 		appcfg, err := config.ReadApplicationConfig()
// 		if err != nil {
// 			csrfMiddleware = csrf.Protect([]byte("default"))
// 			return csrfMiddleware(next)
// 		}

// 		csrfMiddleware = csrf.Protect([]byte(appcfg.Secret))
// 		return csrfMiddleware(next)
// 	}
// }
