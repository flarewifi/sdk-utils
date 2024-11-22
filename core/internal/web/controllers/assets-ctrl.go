package controllers

import (
	"net/http"
	"os"

	"core/internal/plugins"
)

func NewAssetsCtrl(g *plugins.CoreGlobals) *AssetsCtrl {
	return &AssetsCtrl{g}
}

type AssetsCtrl struct {
	g *plugins.CoreGlobals
}

func (ctrl *AssetsCtrl) GetFavicon(w http.ResponseWriter, r *http.Request) {
	contents, err := os.ReadFile(ctrl.g.CoreAPI.Utl.Resource("assets/images/default-favicon-32x32.png"))
	if err != nil {
		ctrl.g.CoreAPI.HttpAPI.HttpResponse().Error(w, r, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(contents)
}
