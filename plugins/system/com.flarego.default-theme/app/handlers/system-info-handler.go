package handlers

import (
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/sysinfo"
	"com.flarego.default-theme/resources/views/admin"
)

func SystemResourceCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get system information
		info, err := sysinfo.GetSystemInfo()
		if err != nil {
			// If there's an error, provide empty/default system info
			info = &sysinfo.SystemInfo{}
		}

		view := admin.ResourceInfo(api, info)
		view.Render(r.Context(), w)
	}
}
