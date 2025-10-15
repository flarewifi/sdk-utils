package adminctrl

import (
	"core/internal/api"
	"core/internal/utils/cmd"
	"net/http"
	"time"
)

func RebootCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(3 * time.Second)
			cmd.Exec("reboot", nil)
		}()

		w.Write([]byte("Rebooting..."))
	}
}

func ShutdownCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(3 * time.Second)
			cmd.Exec("shutdown -h now", nil)
		}()

		w.Write([]byte("Shutting down..."))
	}
}
