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
			res.FlashMsg(w, r, fmt.Sprintf("Unable to remove %v file: %v", filepath.Base(filePath), err), sdkapi.FlashMsgSuccess)
			return
		}

		g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, cookieName)

		res.FlashMsg(w, r, fmt.Sprintf("File %v successfully removed.", filepath.Base(filePath)), sdkapi.FlashMsgSuccess)
	}
}
