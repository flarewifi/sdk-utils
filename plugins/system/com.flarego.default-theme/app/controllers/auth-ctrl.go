package controllers

// import (
// 	"encoding/json"
// 	"net/http"

// 	sdkhttp "sdk/api/http"
// 	sdkplugin "sdk/api/plugin"

// 	"com.flarego.default-theme/resources/views"
// )

// func IndexCtrl(api sdkplugin.PluginApi) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		v := views.IndexPage()
// 		page := sdkhttp.ViewPage{
// 			PageContent: v,
// 		}
// 		api.Http().HttpResponse().PortalView(w, r, page)
// 	}
// }

// func LoginCtrl(api sdkplugin.PluginApi) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		var data struct {
// 			Username string `json:"username"`
// 			Password string `json:"password"`
// 		}

// 		err := json.NewDecoder(r.Body).Decode(&data)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}

// 		// res := api.Http().VueResponse()

// 		acct, err := api.Http().Auth().Authenticate(data.Username, data.Password)
// 		if err != nil {
// 			// res.SendFlashMsg(w, "error", err.Error(), http.StatusUnauthorized)
// 			return
// 		}

// 		if err = api.Http().Auth().SignIn(w, acct); err != nil {
// 			// res.SendFlashMsg(w, "error", err.Error(), http.StatusUnauthorized)
// 			return
// 		}

// 		// res.SendFlashMsg(w, "success", "Logged in successfully.", http.StatusOK)
// 	}
// }

// func LogoutCtrl(api sdkplugin.PluginApi) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		api.Http().Auth().SignOut(w)
// 		// res := api.Http().VueResponse()
// 		// res.SendFlashMsg(w, "info", "You have been logged out.", http.StatusOK)
// 	}
// }
