package adminctrl

import (
	"net/http"

	"core/internal/api"
	sse "core/tools/sse"
)

func AdminSseHandler(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		acct, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(acct.Username(), s)
		s.Listen()
	}
}
