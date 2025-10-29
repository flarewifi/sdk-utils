package adminctrl

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"core/internal/api"
	"core/internal/utils/plugins"

	views "core/resources/views/admin/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func PluginsIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
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

		pluginZipInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.zip")
		pluginGithubInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.github")
		pluginIndexURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.index")
		checkInstallStatusURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.status")

		formRoutes := views.FormRoutes{
			SelectedAction:         pluginGithubInstallURL,
			PluginInstallGithubURL: pluginGithubInstallURL,
			PluginInstallZipURL:    pluginZipInstallURL,
			PluginIndexURL:         pluginIndexURL,
			CheckInstallStatusURL:  checkInstallStatusURL,
		}
		page := views.IndexPage(g.CoreAPI, data, formRoutes)
		view := sdkapi.ViewPage{
			PageContent: page,
			Assets: sdkapi.ViewAssets{
				JsFile: "plugin.js",
			},
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

		pluginZipInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.zip")
		pluginGithubInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.github")
		pluginIndexURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.index")
		checkInstallStatusURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.status")

		page := views.InstallPlugin(api, views.FormRoutes{
			SelectedAction:         pluginGithubInstallURL,
			PluginInstallGithubURL: pluginGithubInstallURL,
			PluginInstallZipURL:    pluginZipInstallURL,
			PluginIndexURL:         pluginIndexURL,
			CheckInstallStatusURL:  checkInstallStatusURL,
		})

		view := sdkapi.ViewPage{
			PageContent: page,
			Assets: sdkapi.ViewAssets{
				JsFile: "plugin.js",
			},
		}

		res.AdminView(w, r, view)
	}
}

func CheckPluginStatusCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installSource := r.URL.Query().Get("source")

		w.Header().Set("Content-Type", "application/json")
		progress := GetStatus(installSource)
		if progress == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  FailedStatus,
				"message": "plugin not found",
			})
			return
		}

		json.NewEncoder(w).Encode(progress)
	}
}

func PluginInstallFromZipCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coreAPI := g.CoreAPI
		res := coreAPI.HttpAPI.Response()
		zipErrorMsg := g.CoreAPI.Translate("error", "zip_install_error")

		admin, err := coreAPI.AcctAPI.Find("admin")
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Parse form (max 10 MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Get uploaded file
		file, header, err := r.FormFile("plugin_zip_file")
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}
		defer file.Close()

		pluginName := header.Filename

		// Save file to uploads directory
		saveDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "uploads")
		if err := sdkutils.FsEnsureDir(saveDir); err != nil {
			UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
			g.CoreAPI.LoggerAPI.Error("zip install error: saving to uploads error: " + err.Error())

			return
		}

		filePath := filepath.Join(saveDir, pluginName)
		out, err := os.Create(filePath)
		if err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			res.FlashMsg(w, r, zipErrorMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		SaveInitialState(pluginName)

		// Launch background installation
		go func(filePath string, filename, pluginName string) {
			UpdateStatus(pluginName, InProgressStatus, "Installing...", 50)

			pluginTmpDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "extracted", sdkutils.RandomStr(16))
			if err := sdkutils.FsExtract(filePath, pluginTmpDir); err != nil {
				UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
				g.CoreAPI.LoggerAPI.Error("zip install error: extract error: " + err.Error())
				return
			}

			pluginSrc, err := sdkutils.FindPluginSrc(pluginTmpDir)
			if err != nil {
				UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
				g.CoreAPI.LoggerAPI.Error("zip install error: find plugin src error: " + err.Error())
				return
			}

			info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
			if err != nil {
				UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
				g.CoreAPI.LoggerAPI.Error("zip install error: get plugins info error: " + err.Error())
				return
			}

			pluginCachePath := filepath.Join(sdkutils.PathPluginCacheDir, info.Package)
			if err := sdkutils.FsCopy(pluginSrc, pluginCachePath); err != nil {
				UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
				g.CoreAPI.LoggerAPI.Error("zip install error: file copy error: " + err.Error())
				return
			}

			def := sdkutils.PluginSrcDef{
				Src:       sdkutils.PluginSrcLocal,
				LocalPath: sdkutils.StripRootPath(pluginCachePath),
			}

			if _, err := plugins.InstallFromLocalPath(g.CoreAPI.SqlDb(), def, plugins.InstallOpts{ForceInstall: false}); err != nil {
				UpdateStatus(pluginName, FailedStatus, zipErrorMsg, 0)
				g.CoreAPI.LoggerAPI.Error("zip install error: install from local path error: " + err.Error())
				return
			}

			UpdateStatus(pluginName, InProgressStatus, "Registering plugin...", 75)
			time.Sleep(5 * time.Second)
			UpdateStatus(pluginName, InProgressStatus, "Adding sample delay", 90)
			time.Sleep(10 * time.Second)

			installPath := plugins.GetInstallPath(info.Package)
			p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
			g.PluginMgr.RegisterPlugin(p)

			successMsg := g.CoreAPI.Translate("info", "plugin_install_success_message", "plugin", pluginName)

			data, err := json.Marshal(map[string]string{
				"success": successMsg,
			})
			if err != nil {
				log.Println("Install Progress json error:", err)
			}

			admin.Emit("install:progress", data)
			UpdateStatus(pluginName, SuccessStatus, successMsg, 100)
		}(filePath, header.Filename, pluginName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status": InProgressStatus,
		})
	}
}

func PluginsInstallFromGitCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		githubErrMsg := g.CoreAPI.Translate("error", "github_install_error")

		admin, err := g.CoreAPI.AcctAPI.Find("admin")
		if err != nil {
			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")

			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		repoURL := r.FormValue("github_repo_url")
		gitRef := r.FormValue("github_ref")

		src, err := sdkutils.ParseGitSource(repoURL)
		if err != nil {
			res.FlashMsg(w, r, githubErrMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")

			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		pluginName := src.Repo
		SaveInitialState(pluginName)

		go func() {
			UpdateStatus(pluginName, InProgressStatus, "Installing...", 50)

			info, err := plugins.InstallFromGitSrc(g.CoreAPI.SqlDb(), sdkutils.PluginSrcDef{
				Src:    sdkutils.PluginSrcGit,
				GitURL: repoURL,
				GitRef: gitRef,
			}, plugins.InstallOpts{ForceInstall: false})
			if err != nil {
				UpdateStatus(pluginName, FailedStatus, githubErrMsg, 0)
				g.CoreAPI.LoggerAPI.Error("InstallFromGitSrc: " + err.Error())

				return
			}

			UpdateStatus(pluginName, InProgressStatus, "Registering plugin...", 75)

			installPath := plugins.GetInstallPath(info.Package)
			p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
			g.PluginMgr.RegisterPlugin(p)

			time.Sleep(5 * time.Second)
			UpdateStatus(pluginName, InProgressStatus, "Adding sample delay", 90)
			time.Sleep(10 * time.Second)

			successMsg := g.CoreAPI.Translate("info", "plugin_install_success_message", "plugin", pluginName)
			UpdateStatus(pluginName, SuccessStatus, successMsg, 100)

			data, err := json.Marshal(map[string]string{
				"success": successMsg,
			})
			if err != nil {
				log.Println("Install Progress JSON error:", err)
			}

			admin.Emit("install:progress", data)
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status": InProgressStatus,
		})
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
