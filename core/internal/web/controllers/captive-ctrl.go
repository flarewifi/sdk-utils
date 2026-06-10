package controllers

import (
	"encoding/json"
	"net/http"

	"core/internal/api"
)

// CaptiveApiCtrl implements the RFC 8908 Captive Portal API, advertised to
// clients via RFC 8910 (DHCP option 114). It returns application/captive+json
// telling the OS whether the client still needs to authenticate and, when it is
// authorized, how much session time remains. Registered on the root router so it
// is reachable at the advertised portal hostname without the LAN-IP redirect.
func CaptiveApiCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/captive+json")
		w.Header().Set("Cache-Control", "no-store")

		userPortalURL := "https://" + r.Host

		resp := map[string]any{
			"captive":                   true,
			"user-portal-url":           userPortalURL,
			"x-android-use-custom-tags": 361335020,
		}

		// A running, non-expired, non-consumed session means the client is online
		// and should be reported as not captive.
		if clnt, err := g.CoreAPI.Http().GetClientDevice(r); err == nil && clnt != nil {
			if sess, ok := g.CoreAPI.SessionsMgr().RunningSession(clnt); ok &&
				sess.IsRunning() && !sess.IsExpired() && !sess.IsConsumed() {
				resp["captive"] = false
				resp["seconds-remaining"] = sess.RemainingTime()
				resp["can-extend-session"] = true
			}
		}

		json.NewEncoder(w).Encode(resp)
	}
}
