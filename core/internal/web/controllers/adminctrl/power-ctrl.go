package adminctrl

import (
	"core/internal/api"
	powerview "core/resources/views/admin/power"
	"net/http"
	sdkapi "sdk/api"
	"time"
	cmd "tools/shell"
)

func RebootPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		page := powerview.RebootPage(g.CoreAPI)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func RebootCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(3 * time.Second)
			cmd.Exec("reboot", nil)
		}()

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="alert alert-success">
			<i class="bi bi-check-circle me-2"></i>
			System is rebooting... Please wait a few minutes before reconnecting.
		</div>`))
	}
}

func ShutdownPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		page := powerview.ShutdownPage(g.CoreAPI)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func ShutdownCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(3 * time.Second)
			cmd.Exec("halt", nil)
		}()

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="alert alert-success">
			<i class="bi bi-check-circle me-2"></i>
			System is shutting down... The device will power off shortly.
		</div>`))
	}
}
