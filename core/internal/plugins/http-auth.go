package plugins

import (
	"errors"
	"net/http"

	"core/internal/config"
	"core/internal/utils/jsonwebtoken"
	webutil "core/internal/utils/web"
	"core/internal/web/helpers"
	sdkacct "sdk/api/accounts"
	sdkhttp "sdk/api/http"
)

func NewHttpAuth(api *PluginApi) *HttpAuth {
	return &HttpAuth{
		api: api,
	}
}

type HttpAuth struct {
	api *PluginApi
}

func (self *HttpAuth) CurrentAcct(r *http.Request) (sdkacct.IAccount, error) {
	return helpers.CurrentAcct(r)
}

func (self *HttpAuth) IsAuthenticated(r *http.Request) bool {
	_, err := webutil.IsAdminAuthenticated(r)
	return err == nil
}

func (self *HttpAuth) Authenticate(username string, password string) (sdkacct.IAccount, error) {
	acct, err := webutil.AuthenticateAdmin(username, password)
	if err != nil {
		err = errors.New(self.api.CoreAPI.Utl.Translate("error", "invalid_login"))
		return nil, err
	}

	return acct, nil
}

func (self *HttpAuth) SignIn(w http.ResponseWriter, acct sdkacct.IAccount) error {
	appcfg, err := config.ReadApplicationConfig()
	if err != nil {
		return err
	}

	payload := map[string]string{"username": acct.Username()}
	token, err := jsonwebtoken.GenerateToken(payload, appcfg.Secret)
	if err != nil {
		return err
	}

	sdkhttp.SetCookie(w, webutil.AuthTokenCookie, token)
	return nil
}

func (self *HttpAuth) SignOut(w http.ResponseWriter) error {
	sdkhttp.SetCookie(w, webutil.AuthTokenCookie, "")
	return nil
}
