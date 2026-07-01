package routes

import (
	sdkapi "sdk/api"

	adminctrl "com.flarego.devkit/app/controllers/admin"
)

func AdminRoutes(api sdkapi.IPluginApi) {
	adminR := api.Http().Router().AdminRouter(nil)

	adminR.Group("/developer", func(r sdkapi.IHttpRouterInstance) {
		r.Get("/", adminctrl.ListCtrl(api)).
			Name("admin:developer:index")

		r.Post("/upload", adminctrl.UploadCtrl(api)).
			Name("admin:developer:upload")

		// pkg is the plugin package id (e.g. com.flarego.example). It is matched
		// greedily because package ids contain dots, which a default path segment
		// would still allow, but {pkg} keeps the route unambiguous next to /upload.
		r.Get("/{pkg}/download", adminctrl.DownloadCtrl(api)).
			Name("admin:developer:download")
	})
}
