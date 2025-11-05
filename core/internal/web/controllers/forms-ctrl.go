package controllers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	sdkapi "sdk/api"

	"core/internal/api"
)

func DeleteFileCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("accessing?")
		res := g.CoreAPI.HttpAPI.Response()

		filePath := r.URL.Query().Get("filepath")
		cookieName := r.URL.Query().Get("cookie_name")
		err := g.CoreAPI.Http().Helpers().RemoveFile(filePath)
		if err != nil {
			log.Println("error removing file: ", err)

			errMsg := g.CoreAPI.Translate("error", "unable_to_remove_file_error", "file", filepath.Base(filePath), "err", err)
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			return
		}

		g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, cookieName)

		successMsg := g.CoreAPI.Translate("info", "remove_file_success_message", "file", filepath.Base(filePath))
		res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
	}
}
