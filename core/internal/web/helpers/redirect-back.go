package helpers

import "net/http"

func RedirectBack(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}
