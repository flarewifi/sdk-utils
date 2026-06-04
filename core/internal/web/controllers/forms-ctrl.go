package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	sdkapi "sdk/api"

	"core/internal/api"
)

func DeleteFileCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		filePath := r.URL.Query().Get("filepath")
		cookieName := r.URL.Query().Get("cookie_name")
		err := os.Remove(filePath)
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Unable to Remove File", "file", filepath.Base(filePath), "err", err)
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			return
		}

		g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, cookieName)

		successMsg := g.CoreAPI.Translate("info", "File Removed Successfully", "file", filepath.Base(filePath))
		res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
	}
}
