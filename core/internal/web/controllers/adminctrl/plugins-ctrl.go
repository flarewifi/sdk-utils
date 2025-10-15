package adminctrl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"core/internal/api"
	"core/internal/utils/plugins"

	views "core/resources/views/admin/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func PluginsIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		allPlugins := g.PluginMgr.All()
		pluginData := []views.PluginData{}
		for _, p := range allPlugins {
			info := p.Info()
			if p.Info().Package != g.CoreAPI.Info().Package {
				def, err := plugins.GetPluginDef(info.Package)
				if err != nil {
					g.CoreAPI.LoggerAPI.Error(err.Error())
					continue
				}
				toBeRemoved := plugins.IsToBeRemoved(info.Package)
				hasPendingUpdate := plugins.HasPendingUpdate(info.Package)
				pluginData = append(pluginData, views.PluginData{
					Info:             info,
					Src:              def,
					ToBeRemoved:      toBeRemoved,
					HasPendingUpdate: hasPendingUpdate,
				})
			}
		}
		data := views.IndexPageData{
			Plugins: pluginData,
		}
		page := views.IndexPage(g.CoreAPI, data)
		view := sdkapi.ViewPage{
			PageContent: page,
		}
		res.AdminView(w, r, view)
	}
}

func DownloadPluginUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()
		vars := api.HttpAPI.MuxVars(r)

		pluginPkg := vars["pkg"]
		tagName := vars["tag"]

		githubErrorMsg := g.CoreAPI.Translate("error", "github_update_error")
		tarballDownloadURL, err := plugins.GetTarballDownloadURL(tagName, pluginPkg)
		if err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		tarball, err := plugins.GetTarballDownloadPath(pluginPkg)
		if err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		if err := sdkutils.DownloadGitHubTarball(tarballDownloadURL, tarball); err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		if err := plugins.CompileDownloadedTarball(tarball, pluginPkg); err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		src, err := sdkutils.FindPluginSrc(filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads", pluginPkg))
		if err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		dst := filepath.Join(sdkutils.PathPluginUpdatesDir, pluginPkg)
		if err := sdkutils.CopyPluginFiles(src, dst); err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Remove the source directory after successfully moving its contents
		if err := plugins.CleanupDownload(); err != nil {
			res.FlashMsg(w, r, githubErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.index")
			g.CoreAPI.LoggerAPI.Error(err.Error())
		}

		githubSuccessMsg := g.CoreAPI.Translate("info", "github__update_success_message")
		res.FlashMsg(w, r, githubSuccessMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}

func CheckPluginUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()

		vars := api.HttpAPI.MuxVars(r)
		pluginPkg := vars["pkg"]

		pluginUpdateErrMsg := g.CoreAPI.Translate("error", "plugin_update_error")

		def, err := plugins.GetPluginDef(pluginPkg)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("get plugin def error: %v", err))
			res.FlashMsg(w, r, pluginUpdateErrMsg, sdkapi.FlashMsgError)

			return
		}

		releases, err := plugins.GetGithubReleases(def.GitURL)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("get plugin def error: %v", err))
			res.FlashMsg(w, r, pluginUpdateErrMsg, sdkapi.FlashMsgError)

			return
		}

		page := views.ReleasesPage(g.CoreAPI, releases, pluginPkg)
		view := sdkapi.ViewPage{
			PageContent: page,
		}
		res.AdminView(w, r, view)
	}
}

func PluginInstallIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		pluginZipInsttallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.zip")
		pluginGithubInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.github")
		pluginIndexURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.index")
		checkInstallStatusURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.status")

		page := views.InstallPlugin(api, views.FormRoutes{
			SelectedAction:         pluginGithubInstallURL,
			PluginInstallGithubURL: pluginGithubInstallURL,
			PluginInstallZipURL:    pluginZipInsttallURL,
			PluginIndexURL:         pluginIndexURL,
			CheckInstallStatusURL:  checkInstallStatusURL,
		})
		view := sdkapi.ViewPage{
			PageContent: page,
			Assets: sdkapi.ViewAssets{
				JsFile: "index.js",
			},
		}
		res.AdminView(w, r, view)
	}
}

func CheckPluginStatusCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("check status called...")

		api := g.CoreAPI
		res := api.Http().Response()
		installSource := r.URL.Query().Get("source")

		w.Header().Set("Content-Type", "application/json")
		plugin := GetPlugin(api, installSource)
		if plugin == nil {
			res.FlashMsg(w, r, "plugin not found", sdkapi.FlashMsgError)

			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  FailedStatus,
				"message": "plugin not found",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"status": plugin.Status,
		})
	}
}

func PluginInstallFromZipCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coreApi := g.CoreAPI
		res := coreApi.HttpAPI.Response()

		zipErrorMsg := g.CoreAPI.Translate("error", "zip_install_error")

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Retrieve the file from the form field "file"
		file, header, err := r.FormFile("plugin_zip_file")
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}
		defer file.Close()

		// Specify the directory where the file will be saved
		saveDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "uploads")
		err = sdkutils.FsEnsureDir(saveDir) // Ensure the directory exists
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Create the full path for the file
		filePath := filepath.Join(saveDir, header.Filename)

		// Create a new file in the specified directory
		dst, err := os.Create(filePath)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}
		defer dst.Close()

		// Copy the uploaded file data to the new file
		_, err = io.Copy(dst, file)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		pluginTmpDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "extracted", sdkutils.RandomStr(16))
		if err = sdkutils.FsExtract(filePath, pluginTmpDir); err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		pluginSrc, err := sdkutils.FindPluginSrc(pluginTmpDir)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		pluginCachePath := filepath.Join(sdkutils.PathPluginCacheDir, info.Package)
		if err = sdkutils.FsCopy(pluginSrc, pluginCachePath); err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		def := sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: sdkutils.StripRootPath(pluginCachePath),
		}

		if _, err := plugins.InstallFromLocalPath(g.CoreAPI.SqlDb(), def); err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		installPath := plugins.GetInstallPath(info.Package)
		p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)

		// Redirect to the plugins index page
		successMsg := g.CoreAPI.Translate("info", "plugin_install_success_message")
		coreApi.HttpAPI.Response().FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}

func PluginsInstallFromGitCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		githubErrMsg := g.CoreAPI.Translate("error", "github_install_error")

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")

			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		repoURL := r.FormValue("github_repo_url")
		gitRef := r.FormValue("github_ref")

		pluginName := getGithubPluginName(repoURL)

		if err := SaveInitialState(g.CoreAPI, pluginName); err != nil {
			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		info, err := plugins.InstallFromGitSrc(g.CoreAPI.SqlDb(), sdkutils.PluginSrcDef{
			Src:    sdkutils.PluginSrcGit,
			GitURL: repoURL,
			GitRef: gitRef,
		})

		if err != nil {
			if err := UpdateStatus(g.CoreAPI, pluginName, FailedStatus); err != nil {
				g.CoreAPI.LoggerAPI.Error("unable to update plugin status: " + err.Error())
			}

			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		installPath := plugins.GetInstallPath(info.Package)
		p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)

		if err := UpdateStatus(g.CoreAPI, pluginName, SuccessStatus); err != nil {
			g.CoreAPI.LoggerAPI.Error("unable to update plugin status: " + err.Error())
		}

		successMsg := g.CoreAPI.Translate("info", "plugin_install_success_message")
		res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}

func UninstallPluginCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()
		vars := api.HttpAPI.MuxVars(r)
		pluginPkg := vars["pkg"]

		uninstallErr := g.CoreAPI.Translate("error", "plugin_uninstall_error")

		err := plugins.MarkToRemove(pluginPkg)
		if err != nil {
			res.FlashMsg(w, r, uninstallErr, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		uninstallMsg := g.CoreAPI.Translate("info", "plugin_uninstall_message")
		res.FlashMsg(w, r, uninstallMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}
