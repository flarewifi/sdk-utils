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
	cookie := &http.Cookie{Name: name, Value: strings.TrimSpace(value), Path: "/", Expires: expires}
	http.SetCookie(w, cookie)
}
