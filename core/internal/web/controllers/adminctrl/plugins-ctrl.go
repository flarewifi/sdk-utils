package adminctrl

import (
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
				pluginData = append(pluginData, views.PluginData{
					Info:        info,
					Src:         def,
					ToBeRemoved: toBeRemoved,
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

func PluginInstallIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		pluginZipInsttallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.zip")
		pluginGithubInstallURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.install.github")
		page := views.InstallPlugin(api, views.FormRoutes{
			SelectedAction:         pluginGithubInstallURL,
			PluginInstallGithubURL: pluginGithubInstallURL,
			PluginInstallZipURL:    pluginZipInsttallURL,
		})
		view := sdkapi.ViewPage{
			PageContent: page,
		}
		res.AdminView(w, r, view)
	}
}

func PluginInstallFromZipCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coreApi := g.CoreAPI
		res := coreApi.HttpAPI.Response()

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		// Retrieve the file from the form field "file"
		file, header, err := r.FormFile("plugin_zip_file")
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}
		defer file.Close()

		// Specify the directory where the file will be saved
		saveDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "uploads")
		err = sdkutils.FsEnsureDir(saveDir) // Ensure the directory exists
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		// Create the full path for the file
		filePath := filepath.Join(saveDir, header.Filename)

		// Create a new file in the specified directory
		dst, err := os.Create(filePath)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}
		defer dst.Close()

		// Copy the uploaded file data to the new file
		_, err = io.Copy(dst, file)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		// Extract the zip file to the plugins/local directory
		pluginTmpDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "extracted", sdkutils.RandomStr(16))
		if err = sdkutils.FsExtract(filePath, pluginTmpDir); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		pluginSrc, err := sdkutils.FindPluginSrc(pluginTmpDir)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		pluginCachePath := filepath.Join(sdkutils.PathAppDir, "plugins", "cache", info.Package)
		if err = sdkutils.FsCopy(pluginSrc, pluginCachePath); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		def := sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: pluginCachePath,
		}

		if _, err := plugins.InstallFromLocalPath(os.Stdout, g.CoreAPI.SqlDb(), def); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		installPath := plugins.GetInstallPath(info.Package)
		p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)

		// Redirect to the plugins index page
		coreApi.HttpAPI.Response().FlashMsg(w, r, "Plugin installed successfully", sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}

func PluginsInstallFromGitCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		repoURL := r.FormValue("github_repo_url")
		gitRef := r.FormValue("github_ref")

		info, err := plugins.InstallFromGitSrc(os.Stdout, g.CoreAPI.SqlDb(), sdkutils.PluginSrcDef{
			Src:    sdkutils.PluginSrcGit,
			GitURL: repoURL,
			GitRef: gitRef,
		})

		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin.plugins.install")
			return
		}

		installPath := plugins.GetInstallPath(info.Package)
		p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)

		res.FlashMsg(w, r, "Plugin installed successfully", sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}

func UninstallPluginCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()
		vars := api.HttpAPI.MuxVars(r)
		pluginPkg := vars["pkg"]

		err := plugins.MarkToRemove(pluginPkg)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		res.FlashMsg(w, r, "Plugin will be removed after the next reboot.", sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin.plugins.index")
	}
}
