package admin

import (
	"net/http"
	"os"
	"strings"

	sdkapi "sdk/api"

	"com.flarego.devkit/app/utils"
)

// DownloadCtrl streams a local plugin's source back to the browser as
// <package>.zip. The archive is built from a cleaned copy of
// data/plugins/local/<package> (no .git, no compiled .so).
func DownloadCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		pkg := strings.TrimSpace(api.Http().MuxVars(r)["pkg"])

		if pkg == "" {
			res.FlashMsg(w, r, api.Translate("error", "No plugin specified"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		zipPath, cleanup, err := utils.ZipSource(pkg)
		defer cleanup()
		if err != nil {
			api.Logger().Error("developer: zip source for " + pkg + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not package the plugin source"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		f, err := os.Open(zipPath)
		if err != nil {
			api.Logger().Error("developer: open zip for " + pkg + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not package the plugin source"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			api.Logger().Error("developer: stat zip for " + pkg + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not package the plugin source"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+pkg+".zip\"")
		http.ServeContent(w, r, pkg+".zip", stat.ModTime(), f)
	}
}
