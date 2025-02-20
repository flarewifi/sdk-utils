package adminctrl

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"core/internal/api"
	"core/internal/utils/pkg"
	views "core/resources/views/admin/plugins"

	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func PluginsIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		plugins := g.PluginMgr.All()
		pluginData := []views.PluginData{}
		for _, p := range plugins {
			info := p.Info()
			if p.Info().Package != g.CoreAPI.Info().Package {
				def, err := pkg.GetPluginDef(info.Package)
				if err != nil {
					g.CoreAPI.LoggerAPI.Error(err.Error())
					continue
				}

				toBeRemoved := pkg.IsToBeRemoved(info.Package)
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

func PluginInstallCtrl(g *api.CoreGlobals) http.HandlerFunc {
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

func PluginInstallFromZip(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coreApi := g.CoreAPI
		res := coreApi.HttpAPI.Response()

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			res.Error(w, r, err, http.StatusBadRequest)
			return
		}

		// Retrieve the file from the form field "file"
		file, header, err := r.FormFile("plugin_zip_file")
		if err != nil {
			http.Error(w, "Failed to get file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Specify the directory where the file will be saved
		saveDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "uploads")
		err = sdkutils.FsEnsureDir(saveDir) // Ensure the directory exists
		if err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Create the full path for the file
		filePath := filepath.Join(saveDir, header.Filename)

		// Create a new file in the specified directory
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to create destination file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file data to the new file
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Extract the zip file to the plugins/local directory
		pluginTmpDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "extracted", sdkutils.RandomStr(16))
		if err = sdkutils.FsExtract(filePath, pluginTmpDir); err != nil {
			http.Error(w, "Failed to extract zip file", http.StatusInternalServerError)
			return
		}

		pluginSrc, err := sdkutils.FindPluginSrc(pluginTmpDir)
		if err != nil {
			http.Error(w, "Failed to extract zip file", http.StatusInternalServerError)
			return
		}

		info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
		if err != nil {
			http.Error(w, "Failed to extract zip file", http.StatusInternalServerError)
			return
		}

		pluginCachePath := filepath.Join(sdkutils.PathAppDir, "plugins", "cache", info.Package)
		if err = sdkutils.FsCopy(pluginSrc, pluginCachePath); err != nil {
			http.Error(w, "Failed to extract zip file", http.StatusInternalServerError)
			return
		}

		def := sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: pluginCachePath,
		}

		if _, err := pkg.InstallFromLocalPath(os.Stdout, g.CoreAPI.SqlDb(), def); err != nil {
			http.Error(w, "Failed to extract zip file", http.StatusInternalServerError)
			return
		}

		installPath := pkg.GetInstallPath(info.Package)
		p := api.NewPluginApi(installPath, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)

		// Redirect to the plugins index page
		coreApi.HttpAPI.Response().FlashMsg(w, r, "Plugin installed successfully", sdkapi.FlashMsgSuccess)
		indexURL := coreApi.HttpAPI.Helpers().UrlForRoute("admin.plugins.index")
		http.Redirect(w, r, indexURL, http.StatusSeeOther)
	}
}

func PluginsStoreIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// srv, ctx := rpc.GetCoreMachineTwirpServiceAndCtx()
		// qPlugins, err := srv.FetchPlugins(ctx, &rpc.FetchPluginsRequest{})
		// if err != nil {
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// if qPlugins == nil {
		// 	err := errors.New("queried plugins is nil")
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// // parse pluginsData
		// var pluginsData []PluginData
		// for _, qP := range qPlugins.Plugins {
		// 	pluginsData = append(pluginsData, PluginData{
		// 		Id: int(qP.PluginId),
		// 		Info: sdkpkg.PluginInfo{
		// 			Name:        qP.Name,
		// 			Package:     qP.Package,
		// 			Description: "",
		// 		},
		// 		IsInstalled: pkg.IsPackageInstalled(qP.Package),
		// 	})
		// }

		// res.Json(w, pluginsData, http.StatusOK)
	}
}

func ViewPluginCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// // parse query
		// pluginId := sdkstr.AtoiOrDefault(r.URL.Query().Get("id"), 0)

		// if pluginId == 0 {
		// 	err := errors.New("invalid plugin id")
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// srv, ctx := rpc.GetCoreMachineTwirpServiceAndCtx()
		// qPlugin, err := srv.FetchPlugin(ctx, &rpc.FetchPluginRequest{
		// 	PluginId: int32(pluginId),
		// })
		// if err != nil {
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// if qPlugin == nil {
		// 	err := errors.New("queried plugin is nil")
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// // parse plugin
		// var pluginReleases []PluginRelease
		// for _, qpr := range qPlugin.Releases {
		// 	pluginReleases = append(pluginReleases, PluginRelease{
		// 		Major:      int(qpr.Major),
		// 		Minor:      int(qpr.Minor),
		// 		Patch:      int(qpr.Patch),
		// 		ZipFileUrl: qpr.ZipFileUrl,
		// 	})
		// }

		// plugin := PluginData{
		// 	Id: int(qPlugin.Plugin.PluginId),
		// 	Info: sdkpkg.PluginInfo{
		// 		Name:        qPlugin.Plugin.Name,
		// 		Package:     qPlugin.Plugin.Package,
		// 		Description: "", // TODO: add the description
		// 	},
		// 	Releases: pluginReleases,
		// }

		// res.Json(w, plugin, http.StatusOK)
	}
}

func UploadFileCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.HttpResponse()

		// // limit file upload size 10 * (2 ** 20) = 10MB
		// if err := r.ParseMultipartForm(10 << 20); err != nil {
		// 	log.Println("Error in parsing multi part form:", err)
		// 	res.Json(w, "", http.StatusInternalServerError)
		// 	return
		// }

		// // get uploaded file
		// uploadedFile, handler, err := r.FormFile("file")
		// if err != nil {
		// 	log.Println("Error in opening form file: ", err)
		// 	res.Json(w, "Error: invalid multipart file", http.StatusInternalServerError)
		// 	return
		// }
		// defer uploadedFile.Close()

		// // prepare parent path
		// parentPath := filepath.Join(sdkpaths.UploadsDir, sdkstr.Rand(6))

		// // ensure parent directory exists
		// if err := os.MkdirAll(parentPath, 0755); err != nil {
		// 	log.Println("Error creating parent dir:", err)
		// 	res.Json(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// // create destination file
		// filePath := filepath.Join(parentPath, handler.Filename)
		// prZipFile, err := os.Create(filePath)
		// if err != nil {
		// 	log.Println("Error creating pr zip file:", err)
		// 	res.Json(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// defer prZipFile.Close()

		// // copy the contents of the uploaded file on to the created destination file
		// if _, err := io.Copy(prZipFile, uploadedFile); err != nil {
		// 	log.Println("Error copying file:", err)
		// 	res.Json(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// log.Printf("%s successfully uploaded", filePath)
		// res.Json(w, filePath, http.StatusOK)
	}
}

// TODO: update for multiple files for future use-case
func UploadFilesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.HttpResponse()

		// // TODO: implementation of multiple file uploads

		// res.Json(w, "", http.StatusOK)
	}
}

func PluginsInstallCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// // read post body as json
		// var reqDef pkg.PluginSrcDef
		// err := json.NewDecoder(r.Body).Decode(&reqDef)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// var result strings.Builder
		// info, err := pkg.InstallSrcDef(&result, reqDef)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// res.Json(w, info, http.StatusOK)
	}
}

// func getInstalledPlugins() []PluginData {
// sources := pkg.InstalledPluginsList()
// plugins := []PluginData{}

// for _, def := range sources {
// 	info, err := pkg.GetInfoFromDef(def)
// 	if err != nil {
// 		return nil
// 	}

// 	p := PluginData{
// 		Info:             info,
// 		Src:              def,
// 		HasPendingUpdate: pkg.HasPendingUpdate(info.Package),
// 		ToBeRemoved:      pkg.IsToBeRemoved(info.Package),
// 	}

// 	plugins = append(plugins, p)
// }

// return plugins
// }

func UninstallPluginCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()
		vars := api.HttpAPI.MuxVars(r)
		pluginPkg := vars["pkg"]

		err := pkg.MarkToRemove(pluginPkg)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		api.HttpAPI.Response().FlashMsg(w, r, "Plugin will be removed after the next reboot.", sdkapi.FlashMsgSuccess)
		indexURL := api.HttpAPI.Helpers().UrlForRoute("admin.plugins.index")
		http.Redirect(w, r, indexURL, http.StatusSeeOther)
	}
}

func UpdatePluginCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// // read post body as json
		// var def pkg.PluginSrcDef
		// err := json.NewDecoder(r.Body).Decode(&def)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// var result strings.Builder
		// info, err := pkg.InstallSrcDef(&result, def)
		// if err != nil {
		// 	log.Println("Error updating/installing source from def:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// res.Json(w, info, http.StatusOK)
	}
}

func CheckPluginUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// pluginsInstallData := pkg.InstalledPluginsList()
		// var pluginsResponseData []PluginData

		// for i, pInstallDatum := range pluginsInstallData {
		// 	pInfo, err := pkg.GetPluginInfo(pInstallDatum.Def)
		// 	if err != nil {
		// 		log.Println("Error reading plugin info:", err)
		// 		res.Error(w, err.Error(), http.StatusBadRequest)
		// 		return
		// 	}

		// 	hasUpdates, err := updates.CheckForPluginUpdates(&pInstallDatum, pInfo)
		// 	if err != nil {
		// 		log.Println("Error checking updates:", err)
		// 		res.Error(w, err.Error(), http.StatusBadRequest)
		// 		return
		// 	}

		// 	pluginsResponseData = append(pluginsResponseData, PluginData{
		// 		Id:               i,
		// 		Info:             pInfo,
		// 		Src:              pInstallDatum,
		// 		HasPendingUpdate: pkg.HasPendingUpdate(pInfo.Package),
		// 		HasUpdates:       hasUpdates,
		// 		ToBeRemoved:      false,
		// 		IsInstalled:      true,
		// 		Releases:         []PluginRelease{},
		// 	})
		// }

		// res.Json(w, pluginsResponseData, http.StatusOK)
	}
}
