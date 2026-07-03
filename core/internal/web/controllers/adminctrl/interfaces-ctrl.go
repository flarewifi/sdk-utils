package adminctrl

import (
	"fmt"
	"net/http"
	"strconv"

	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/network"
	interfacesview "core/resources/views/admin/interfaces"
	"core/utils/config"
)

// InterfacesIndexCtrl renders the Interfaces page: one card per managed LAN,
// where the admin picks the main interface (its IP hosts the portal / custom
// domain), toggles the captive portal per interface, and — for captive
// interfaces — configures the inline bandwidth form.
func InterfacesIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		params := buildInterfacesParams(g)
		page := interfacesview.AdminInterfacesIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

// InterfacesSaveCtrl persists the main interface, per-interface captive/open
// mode, and the inline bandwidth settings, then applies everything live:
// bandwidth via Config().Bandwidth().Save (updates running sessions) and the
// portal firewall + DNS via network.ApplyPortalConfig.
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

		// Validate the main interface: it must be one of the managed LANs.
		mainIf := r.PostFormValue("main_interface")
		if !isKnownLan(lans, mainIf) {
			redirectErr(g.CoreAPI.Translate("error", "Select a valid main interface"))
			return
		}

		// Start from the existing config so any future WAN entries / policy are
		// preserved; we only rewrite the LAN roles here.
		cfg, _ := config.ReadInterfacesConfig()
		cfg.MainInterface = mainIf

		// Collect the bandwidth saves and apply them only after all validation
		// passes, so a bad value on the last interface doesn't leave a half-applied
		// state.
		type pendingBw struct {
			ifname string
			cfg    sdkapi.IBandwdCfg
		}
		var bwSaves []pendingBw

		for _, lan := range lans {
			managed := r.PostFormValue("managed_"+lan.Name) != ""
			// Captive requires managed — the redirect only applies to managed
			// interfaces, so ignore a stray captive flag when management is off.
			captive := managed && r.PostFormValue("captive_"+lan.Name) != ""
			cfg.Interfaces[lan.Name] = config.InterfaceCfg{
				Role:          config.RoleLan,
				Managed:       managed,
				CaptivePortal: captive,
			}

			// Bandwidth settings are required for managed interfaces (they serve
			// sessions); an unmanaged interface is left untouched, so we skip it.
			if !managed {
				continue
			}

			bw, ferr := parseBandwidthForm(r, lan.Name)
			if ferr != "" {
				redirectErr(g.CoreAPI.Translate("error", ferr))
				return
			}
			bwSaves = append(bwSaves, pendingBw{ifname: lan.Name, cfg: bw})
		}

		// Persist the interface roles first.
		if err := config.WriteInterfacesConfig(cfg); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Unable to save interface settings"))
			return
		}

		// Apply bandwidth (persists bandwidth.json + updates running sessions).
		for _, s := range bwSaves {
			if err := g.CoreAPI.Config().Bandwidth().Save(s.ifname, s.cfg); err != nil {
				g.CoreAPI.LoggerAPI.Error(err.Error())
				redirectErr(g.CoreAPI.Translate("error", "Unable to save bandwidth settings"))
				return
			}
		}

		// Apply the portal firewall + DNS live (captive vs open per interface, and
		// the shared DNAT target = the main interface IP).
		if err := network.ApplyPortalConfig(); err != nil {
			g.CoreAPI.LoggerAPI.Error(err.Error())
			redirectErr(g.CoreAPI.Translate("error", "Settings saved but could not be applied to the network"))
			return
		}

		res.FlashMsg(w, r, g.CoreAPI.Translate("info", "Interface settings saved"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, indexURL, http.StatusSeeOther)
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// buildInterfacesParams merges the live LAN list with interfaces.json + the
// per-interface bandwidth config into the view params.
func buildInterfacesParams(g *api.CoreGlobals) interfacesview.AdminInterfacesIndexParams {
	lans := network.ListLanInfos()
	cfg, _ := config.ReadInterfacesConfig()
	mainIf := network.MainInterface() // resolved (config or first LAN)

	rows := make([]interfacesview.InterfaceRow, 0, len(lans))
	for _, lan := range lans {
		managed := cfg.IsManaged(lan.Name, lan.Device)
		captive := cfg.IsCaptive(lan.Name, lan.Device)

		// Bandwidth: existing config, or the "no cap" default (UseGlobal, 0/0) for
		// a newly-managed interface with no bandwidth entry yet.
		bw := interfacesview.BandwidthForm{UseGlobal: true}
		if existing, ok := g.CoreAPI.Config().Bandwidth().Get(lan.Name); ok {
			bw = interfacesview.BandwidthForm{
				UseGlobal:  existing.UseGlobal,
				GlobalDown: existing.GlobalDownMbits,
				GlobalUp:   existing.GlobalUpMbits,
				UserDown:   existing.UserDownMbits,
				UserUp:     existing.UserUpMbits,
			}
		}

		rows = append(rows, interfacesview.InterfaceRow{
			Name:      lan.Name,
			Device:    lan.Device,
			IPv4:      lan.IPv4,
			CIDR:      lan.CIDR,
			Up:        lan.Up,
			IsMain:    lan.Name == mainIf,
			Managed:   managed,
			Captive:   captive,
			XData:     fmt.Sprintf("{ managed: %t, captive: %t }", managed, captive),
			Bandwidth: bw,
		})
	}

	return interfacesview.AdminInterfacesIndexParams{
		Rows:      rows,
		MainIP:    mainIP(lans, mainIf),
	}
}

// parseBandwidthForm reads and validates the inline bandwidth fields for ifname.
// Returns a non-empty error message string on the first invalid field.
func parseBandwidthForm(r *http.Request, ifname string) (sdkapi.IBandwdCfg, string) {
	useGlobal := r.PostFormValue("use_global_"+ifname) != ""

	globalDown, err := parseMbits(r.PostFormValue("global_down_" + ifname))
	if err != "" {
		return sdkapi.IBandwdCfg{}, err
	}
	globalUp, err := parseMbits(r.PostFormValue("global_up_" + ifname))
	if err != "" {
		return sdkapi.IBandwdCfg{}, err
	}
	userDown, err := parseMbits(r.PostFormValue("user_down_" + ifname))
	if err != "" {
		return sdkapi.IBandwdCfg{}, err
	}
	userUp, err := parseMbits(r.PostFormValue("user_up_" + ifname))
	if err != "" {
		return sdkapi.IBandwdCfg{}, err
	}

	return sdkapi.IBandwdCfg{
		UseGlobal:       useGlobal,
		GlobalDownMbits: globalDown,
		GlobalUpMbits:   globalUp,
		UserDownMbits:   userDown,
		UserUpMbits:     userUp,
	}, ""
}

// parseMbits parses a required, non-negative Mbps value. An empty string is 0
// (unlimited / auto-detect). Returns an error message on an invalid value.
func parseMbits(s string) (int, string) {
	if s == "" {
		return 0, ""
	}
	n, convErr := strconv.Atoi(s)
	if convErr != nil || n < 0 {
		return 0, "Bandwidth values must be whole numbers of 0 or more"
	}
	return n, ""
}

func isKnownLan(lans []network.LanInfo, name string) bool {
	for _, l := range lans {
		if l.Name == name {
			return true
		}
	}
	return false
}

func mainIP(lans []network.LanInfo, mainIf string) string {
	for _, l := range lans {
		if l.Name == mainIf {
			return l.IPv4
		}
	}
	return ""
}
