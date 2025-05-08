package controllers

import (
	"errors"
	"net/http"
	"os"

	"core/internal/api"
)

func NewAssetsCtrl(g *api.CoreGlobals) *AssetsCtrl {
	return &AssetsCtrl{g}
}

type AssetsCtrl struct {
	g *api.CoreGlobals
}

func (ctrl *AssetsCtrl) GetFavicon(w http.ResponseWriter, r *http.Request) {
	readAssetsError := errors.New(ctrl.g.CoreAPI.Translate("error", "read_assets_error"))
	contents, err := os.ReadFile(ctrl.g.CoreAPI.Utl.Resource("assets/images/default-favicon-32x32.png"))
	if err != nil {
		ctrl.g.CoreAPI.HttpAPI.Response().Error(w, r, readAssetsError, http.StatusInternalServerError)
		ctrl.g.CoreAPI.LoggerAPI.Error(err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(contents)
}
