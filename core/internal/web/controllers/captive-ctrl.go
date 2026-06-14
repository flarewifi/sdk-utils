package controllers

import (
	"encoding/json"
	"net/http"

	"core/internal/api"
	"core/utils/hostfinder"

	sdkapi "sdk/api"
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
			"x-android-use-custom-tabs": 361335020,
		}

		// A hit on this endpoint means the client's OS parsed DHCP option 114 and is
		// probing the RFC 8908 API, i.e. the device is provably active on the network.
		// Resolve the device so we can (a) report remaining session time and (b) wire the
		// probe into the core device-connection tracking process by emitting
		// EventClientActive — subscribers (e.g. the wifi-hotspot auto-resume handler)
		// treat it like a (re)connection. The emit is async and fires on every probe;
		// downstream resume logic is idempotent.
		clnt := resolveCaptiveDevice(g, r)
		if clnt != nil {
			g.EventsMgr.EmitClientEvent(sdkapi.EventClientActive, clnt)

			// A running, non-expired, non-consumed session means the client is online
			// and should be reported as not captive.
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

// resolveCaptiveDevice identifies the device behind a captive-portal probe.
//
// The OS captive-detection agent (CNA) typically fetches /api/captive without the
// portal's device cookie/token (different cookie scope, no app JS), so the normal
// token-based lookup fails for exactly the probes we most want to track. We fall
// back to identifying the device by its source IP→MAC, the same way portal
// registration does. Returns nil if the device cannot be identified.
func resolveCaptiveDevice(g *api.CoreGlobals, r *http.Request) sdkapi.IClientDevice {
	// Priority 1: authenticated device (cookie / X-Device-Token).
	if clnt, err := g.CoreAPI.Http().GetClientDevice(r); err == nil && clnt != nil {
		return clnt
	}

	// Priority 2: fall back to source IP/MAC for unauthenticated OS probes.
	h, err := hostfinder.GetHostFromRequest(r)
	if err != nil || h == nil || h.MacAddr == "" {
		return nil
	}
	if clnt, err := g.CoreAPI.SessionsMgr().FindClientByMac(r.Context(), h.MacAddr); err == nil && clnt != nil {
		return clnt
	}
	return nil
}
