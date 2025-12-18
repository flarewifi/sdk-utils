package api

import (
	"errors"
	"net/http"
	"strings"

	"core/internal/accounts"
	"core/tools/jsonwebtoken"
	sdkapi "sdk/api"
	"core/tools/config"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AUTH_TOKEN_COOKIE = "auth_token"
)

func NewHttpAuth(api *PluginApi) *HttpAuth {
	return &HttpAuth{
		api: api,
	}
}

type HttpAuth struct {
	api *PluginApi
}

func (self *HttpAuth) CurrentAcct(r *http.Request) (sdkapi.IAccount, error) {
	sym := r.Context().Value(sdkapi.SysAcctCtxKey)
	acct, ok := sym.(*accounts.Account)
	if !ok {
		return nil, errors.New("Can't determine current admin account.")
	}

	return acct, nil
}

func (self *HttpAuth) IsAuthenticated(r *http.Request) (sdkapi.IAccount, error) {
	authtoken, err := self.api.CoreAPI.HttpAPI.Cookie().GetCookie(r, AUTH_TOKEN_COOKIE)
	if err != nil {
		// Try header if cookie not found
		bearer := r.Header.Get("Authorization")
		splitToken := strings.Split(bearer, "Bearer ")
		if len(splitToken) != 2 {
			return nil, errors.New("invalid auth token")
		}

		authtoken = splitToken[1]
	}

	appcfg, err := config.ReadApplicationConfig()
	if err != nil {
		return nil, err
	}

	token, err := jsonwebtoken.VerifyToken(authtoken, appcfg.Secret)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, errors.New("invalid jwt claims")
	}

	username := claims["username"].(string)

	return accounts.Find(username)
}

func (self *HttpAuth) Authenticate(username string, password string) (sdkapi.IAccount, error) {
	ErrAuthenticationFailed := errors.New(self.api.CoreAPI.Translate("error", "Authentication failed"))

	acct, err := accounts.Find(username)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}

	if !acct.Auth(password) {
		return nil, ErrAuthenticationFailed
	}

	return acct, nil
}

func (self *HttpAuth) SignIn(w http.ResponseWriter, acct sdkapi.IAccount) error {
	appcfg, err := config.ReadApplicationConfig()
	if err != nil {
		return err
	}

	payload := map[string]string{"username": acct.Username()}
	token, err := jsonwebtoken.GenerateToken(payload, appcfg.Secret)
	if err != nil {
		return err
	}

	self.api.CoreAPI.HttpAPI.Cookie().SetCookie(w, AUTH_TOKEN_COOKIE, token, nil)
	return nil
}

func (self *HttpAuth) SignOut(w http.ResponseWriter) error {
	self.api.CoreAPI.HttpAPI.Cookie().SetCookie(w, AUTH_TOKEN_COOKIE, "", nil)
	return nil
}
