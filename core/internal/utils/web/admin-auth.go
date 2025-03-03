package webutil

// import (
// 	"core/internal/accounts"
// 	"core/internal/config"
// 	"core/internal/utils/jsonwebtoken"
// 	"errors"
// 	"net/http"
// 	sdkhttp "sdk/api/http"
// 	"strings"

// 	"github.com/golang-jwt/jwt/v5"
// )

// const (
// 	AuthTokenCookie = "auth-token"
// )

// var (
// 	ErrAuthenticationFailed = errors.New("authentication failed")
// )

// func IsAdminAuthenticated(api *PluginApi, r *http.Request) (*accounts.Account, error) {
// 	authtoken, err := sdkhttp.GetCookie(r, AuthTokenCookie)
// 	if err != nil {
// 		bearer := r.Header.Get("Authorization")
// 		splitToken := strings.Split(bearer, "Bearer ")
// 		if len(splitToken) != 2 {
// 			return nil, errors.New("invalid auth token")
// 		}

// 		authtoken = splitToken[1]
// 	}

// 	appcfg, err := config.ReadApplicationConfig()
// 	if err != nil {
// 		return nil, err
// 	}

// 	token, err := jsonwebtoken.VerifyToken(authtoken, appcfg.Secret)
// 	if err != nil {
// 		return nil, err
// 	}

// 	claims, ok := token.Claims.(jwt.MapClaims)
// 	if !(ok && token.Valid) {
// 		return nil, errors.New("invalid jwt claims")
// 	}

// 	username := claims["username"].(string)

// 	return accounts.Find(username)
// }

// func AuthenticateAdmin(username string, password string) (*accounts.Account, error) {
// 	acct, err := accounts.Find(username)
// 	if err != nil {
// 		return nil, ErrAuthenticationFailed
// 	}

// 	if !acct.Auth(password) {
// 		return nil, ErrAuthenticationFailed
// 	}

// 	return acct, nil
// }
