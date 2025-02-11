package api

import (
	"net/http"
	"strings"
	"time"
)

func NewHttpCookie() *HttpCookie {
	return &HttpCookie{}
}

type HttpCookie struct{}

func (self *HttpCookie) GetCookie(r *http.Request, name string) (value string, err error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (self *HttpCookie) SetCookie(w http.ResponseWriter, name string, value string) {
	expires := time.Now().AddDate(1, 0, 0)
	cookie := &http.Cookie{
		Name:  name,
		Value: strings.TrimSpace(value),
		Path:  "/", Expires: expires,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

// DeleteCookie deletes the cookie value for a given cookie name
func (self *HttpCookie) DeleteCookie(w http.ResponseWriter, name string) {
	expires := time.Now().AddDate(0, 0, -1)
	cookie := &http.Cookie{Name: name, Value: "", Path: "/", Expires: expires}
	http.SetCookie(w, cookie)
}
