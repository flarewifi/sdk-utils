package adminctrl

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/modules/uci"
	"core/internal/network"
	interfacesview "core/resources/views/admin/interfaces"
	"core/utils/config"
	cmd "core/utils/shell"
)

// InterfacesIndexCtrl renders the Interfaces page: one card per LAN candidate,
// where the admin picks the portal interface (its IP hosts the portal / custom
// domain), toggles the captive portal per interface, and — for captive
// interfaces — configures the static IP. Bandwidth is set separately (in the
// wifi-hotspot plugin's Bandwidth page).
func InterfacesIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		params := buildInterfacesParams(g)
		page := interfacesview.AdminInterfacesIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

// InterfacesSaveCtrl persists the portal interface, the per-interface captive
// flag, and the static IP/netmask, then applies the config live (TC + portal
// firewall + DNS) via network.ReconcileInterfaces. The static IP is only stored
// here — it is pushed to the OS by InterfacesApplyCtrl.
func InterfacesSaveCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		indexURL := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:interfaces:index")

		redirectErr := func(msg string) {
			res.FlashMsg(w, r, msg, sdkapi.FlashMsgError)
			http.Redirect(w, r, indexURL, http.StatusSeeOther)
		}

		if err := r.ParseForm(); err != nil {
			redirectErr(g.CoreAPI.Translate("error", "Could not read the submitted form"))
			return
		}

		lans := network.ListLanInfos()
		if len(lans) == 0 {
			redirectErr(g.CoreAPI.Translate("error", "No LAN interfaces are available to configure"))
			return
		}

		portalIf := r.PostFormValue("portal_interface")

		cfg, _ := config.ReadInterfacesConfig()
		// Rewrite the LAN map wholesale from the submitted form.
		cfg.LanInterfaces = map[string]config.LanInterfaceCfg{}

		captiveCount := 0
		for _, lan := range lans {
			enable := r.PostFormValue("enable_captive_portal_"+lan.Name) != ""

			ip := strings.TrimSpace(r.PostFormValue("ip_" + lan.Name))
			mask := strings.TrimSpace(r.PostFormValue("netmask_" + lan.Name))
			if verr := validateStaticAddr(ip, mask); verr != "" {
				redirectErr(g.CoreAPI.Translate("error", verr))
				return
			}

			cfg.LanInterfaces[lan.Name] = config.LanInterfaceCfg{
				EnableCaptivePortal: enable,
				IpAddress:           ip,
				Netmask:             mask,
			}

			if enable {
				captiveCount++
			}
		}

		// The portal interface hosts the captive portal and is the shared DNAT +
		// DNS target, so it must itself be a captive interface (registered and
		// resolvable). When nothing is captive there is no portal to host.
		if captiveCount > 0 {
			if pic, ok := cfg.LanInterfaces[portalIf]; !ok || !pic.EnableCaptivePortal {
				redirectErr(g.CoreAPI.Translate("error", "The portal interface must be one with the captive portal enabled"))
				return
			}
			cfg.PortalInterface = portalIf
		} else {
			cfg.PortalInterface = ""
		}

		// Persist the interface config first.
		if err := config.WriteInterfacesConfig(cfg); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Unable to save interface settings"))
			return
		}

		// Apply live: set up / tear down TC per the captive toggle and push the
		// portal firewall + DNS (captive vs free per interface, shared DNAT target =
		// the portal interface IP). Reconcile makes a captive toggle take effect
		// without a restart.
		if err := network.ReconcileInterfaces(); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Settings saved but could not be applied to the network"))
			return
		}

		res.FlashMsg(w, r, g.CoreAPI.Translate("info", "Interface settings saved"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, indexURL, http.StatusSeeOther)
	}
}

// InterfacesApplyCtrl writes the stored static IP/netmask of each captive
// interface to the OS (/etc/config/network via UCI), reloads netifd, then
// re-applies the portal config so the new addresses take effect. This is the
// deliberate, OS-mutating counterpart to Save (which only persists config).
func InterfacesApplyCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		indexURL := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:interfaces:index")

		redirectErr := func(msg string) {
			res.FlashMsg(w, r, msg, sdkapi.FlashMsgError)
			http.Redirect(w, r, indexURL, http.StatusSeeOther)
		}

		cfg, _ := config.ReadInterfacesConfig()
		uciNet := uci.NewUciNetworkApi()

		applied := 0
		for _, lan := range network.ListLanInfos() {
			ic, ok := cfg.LanInterfaces[lan.Name]
			// Only captive interfaces with an explicit static address are pushed.
			if !ok || !ic.EnableCaptivePortal || ic.IpAddress == "" || ic.Netmask == "" {
				continue
			}
			if err := uciNet.SetInterface(lan.Name, &sdkapi.INetIface{
				Section: lan.Name,
				Device:  lan.Device,
				Proto:   "static",
				IpAddr:  ic.IpAddress,
				Netmask: ic.Netmask,
			}); err != nil {
				g.CoreAPI.LoggerAPI.Error(err.Error())
				redirectErr(g.CoreAPI.Translate("error", "Unable to apply the IP address to an interface"))
				return
			}
			applied++
		}

		if applied == 0 {
			res.FlashMsg(w, r, g.CoreAPI.Translate("info", "No static IP addresses to apply"), sdkapi.FlashMsgSuccess)
			http.Redirect(w, r, indexURL, http.StatusSeeOther)
			return
		}

		if err := uci.UciTree.Commit(); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Unable to commit network configuration"))
			return
		}

		// Reload netifd so the new addresses take effect. The resulting ifup events
		// drive the registry/CIDR refresh + TC reinit; ReconcileInterfaces re-points
		// the DNAT/DNS at the (possibly changed) portal IP and syncs TC state as a
		// belt-and-suspenders.
		if err := cmd.Exec("/etc/init.d/network reload", nil); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Applied settings but could not reload the network"))
			return
		}
		_ = network.ReconcileInterfaces()

		res.FlashMsg(w, r, g.CoreAPI.Translate("info", "Network settings applied"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, indexURL, http.StatusSeeOther)
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// buildInterfacesParams merges the live LAN list with interfaces.json into the
// view params. Bandwidth is configured separately (wifi-hotspot plugin).
func buildInterfacesParams(g *api.CoreGlobals) interfacesview.AdminInterfacesIndexParams {
	lans := network.ListLanInfos()
	cfg, _ := config.ReadInterfacesConfig()
	portalIf := network.MainInterface() // resolved (config or first captive LAN)

	rows := make([]interfacesview.InterfaceRow, 0, len(lans))
	for _, lan := range lans {
		enable := cfg.IsCaptivePortalEnabled(lan.Name, lan.Device)
		ic := cfg.LanInterfaces[lan.Name] // zero value if no explicit entry

		rows = append(rows, interfacesview.InterfaceRow{
			Name:                lan.Name,
			Device:              lan.Device,
			IPv4:                lan.IPv4,
			CIDR:                lan.CIDR,
			Up:                  lan.Up,
			IsMain:              lan.Name == portalIf,
			EnableCaptivePortal: enable,
			IpAddress:           ic.IpAddress,
			Netmask:             ic.Netmask,
			XData:               fmt.Sprintf("{ enableCaptivePortal: %t }", enable),
		})
	}

	return interfacesview.AdminInterfacesIndexParams{
		Rows:   rows,
		MainIP: mainIP(lans, portalIf),
	}
}

// validateStaticAddr checks an optional static IP + netmask pair. Both empty is
// fine (the interface keeps its current / DHCP address). Returns an error message
// string on an invalid value.
func validateStaticAddr(ip, mask string) string {
	if ip == "" && mask == "" {
		return ""
	}
	if ip == "" || mask == "" {
		return "Provide both an IP address and a netmask, or leave both blank"
	}
	if net.ParseIP(ip) == nil {
		return "Enter a valid IP address"
	}
	if !isValidIPv4Mask(mask) {
		return "Enter a valid netmask (e.g. 255.255.255.0)"
	}
	return ""
}

// isValidIPv4Mask reports whether s is a valid, canonical (contiguous)
// dotted-decimal IPv4 netmask such as 255.255.255.0.
func isValidIPv4Mask(s string) bool {
	v4 := net.ParseIP(s).To4()
	if v4 == nil {
		return false
	}
	// Size() returns (0, 0) for a non-canonical (non-contiguous) mask.
	_, bits := net.IPv4Mask(v4[0], v4[1], v4[2], v4[3]).Size()
	return bits == 32
}

func mainIP(lans []network.LanInfo, mainIf string) string {
	for _, l := range lans {
		if l.Name == mainIf {
			return l.IPv4
		}
	}
	return ""
}
