package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"core/internal/accounts"
	"core/utils/config"
	"core/utils/jsonwebtoken"
	sdkapi "sdk/api"

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

	var foundAcct *accounts.Account
	token, err := jwt.Parse(authtoken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("invalid claims")
		}
		username, ok := claims["username"].(string)
		if !ok || username == "" {
			return nil, errors.New("missing username in token")
		}
		acct, err := accounts.Find(username)
		if err != nil {
			return nil, err
		}
		foundAcct = acct
		return []byte(appcfg.Secret + acct.PasswdHash), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid auth token")
	}

	return foundAcct, nil
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

	acct, err = accounts.Find(username)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}

	return acct, nil
}

func (self *HttpAuth) SignIn(w http.ResponseWriter, acct sdkapi.IAccount) error {
	appcfg, err := config.ReadApplicationConfig()
	if err != nil {
		return err
	}

	a, ok := acct.(*accounts.Account)
	if !ok {
		return errors.New("unsupported account type")
	}

	signingKey := appcfg.Secret + a.PasswdHash
	payload := map[string]string{"username": acct.Username()}
	token, err := jsonwebtoken.GenerateToken(payload, signingKey)
	if err != nil {
		return err
	}

	self.api.CoreAPI.HttpAPI.Cookie().SetCookie(w, AUTH_TOKEN_COOKIE, token, nil)
	return nil
}

func (self *HttpAuth) SignOut(w http.ResponseWriter) error {
	self.api.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, AUTH_TOKEN_COOKIE)
	return nil
}
