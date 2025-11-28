package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	sdkapi "sdk/api"
)

func NewHttpCookie(api *PluginApi) *HttpCookie {
	return &HttpCookie{api}
}

type HttpCookie struct {
	api *PluginApi
}

func (self *HttpCookie) GetCookie(r *http.Request, name string) (value string, err error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	machineID := self.api.Machine().GetID()
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(machineID), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	val, ok := claims["value"].(string)
	if !ok {
		return "", fmt.Errorf("invalid value claim")
	}
	return val, nil
}

func (self *HttpCookie) SetCookie(w http.ResponseWriter, name string, value string, opts *sdkapi.HttpCookieOpts) {
	machineID := self.api.Machine().GetID()
	claims := jwt.MapClaims{
		"value": strings.TrimSpace(value),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedValue, err := token.SignedString([]byte(machineID))
	if err != nil {
		// Handle error - for now, set empty value
		signedValue = ""
	}
	cookie := &http.Cookie{
		Name:  name,
		Value: signedValue,
	}
	if opts != nil {
		cookie.Path = opts.Path
		cookie.Expires = opts.Expires
		cookie.SameSite = opts.SameSite
	} else {
		cookie.Path = "/"
		cookie.Expires = time.Now().Add(24 * time.Hour)
		cookie.SameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, cookie)
}

// DeleteCookie deletes the cookie value for a given cookie name
func (self *HttpCookie) DeleteCookie(w http.ResponseWriter, name string) {
	expires := time.Now().AddDate(0, 0, -1)
	cookie := &http.Cookie{Name: name, Value: "", Path: "/", Expires: expires}
	http.SetCookie(w, cookie)
}
