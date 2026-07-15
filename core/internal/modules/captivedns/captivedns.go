// Package captivedns writes the dnsmasq configuration that makes the shared
// captive-portal hostname resolve locally (split-horizon DNS) and advertises the
// RFC 8908 Captive Portal API to clients via the RFC 8910 DHCP option 114.
package captivedns

import (
	"fmt"
	"strings"

	gouci "github.com/digineo/go-uci"

	"core/internal/modules/uci"
	"core/utils/env"
	cmd "core/utils/shell"
)

// Setup points the captive-portal hostname at the router's main LAN IP for
// connected clients (split-horizon) and advertises the RFC 8908 captive-portal
// API URL via DHCP option 114 on the DHCP pool of EVERY captive interface —
// clients on any captive subnet get the RFC 8910 advertisement, not just the
// main LAN's. Pools of interfaces that are NOT captive have any stale option-114
// entry removed, so toggling an interface free on the Interfaces page stops
// advertising the portal to its clients. Pools are resolved by each UCI dhcp
// section's `interface` option (never by section name); an interface with no
// DHCP pool is skipped.
//
// It is idempotent: prior entries for the same domain and for option 114 are
// replaced rather than duplicated, then dnsmasq is reloaded once. A build with
// no portal domain (dev/devkit) or a missing main LAN IP is a no-op (nothing to
// advertise).
func Setup(lanIP string, captiveIfnames []string, freeIfnames []string) error {
	// The portal hostname is the cloud-issued portal domain, derived from the build
	// environment (env.PortalDomain). When it is empty (dev/devkit) there is no
	// portal host to advertise, so split-horizon DNS and the DHCP option-114 portal
	// URL are skipped (the no-op below). Set on staging and prod.
	domain := env.PortalDomain()
	if domain == "" || lanIP == "" {
		return nil
	}

	if err := setSplitHorizon(domain, lanIP); err != nil {
		return err
	}
	for _, ifname := range captiveIfnames {
		if err := setCaptivePortalOption(domain, ifname, true); err != nil {
			return err
		}
	}
	for _, ifname := range freeIfnames {
		if err := setCaptivePortalOption(domain, ifname, false); err != nil {
			return err
		}
	}

	if err := uci.UciTree.Commit(); err != nil {
		return fmt.Errorf("uci commit dhcp: %w", err)
	}

	// Reload (not restart) so existing DHCP leases are preserved.
	if err := cmd.Exec("service dnsmasq reload", nil); err != nil {
		return fmt.Errorf("dnsmasq reload: %w", err)
	}
	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// setSplitHorizon ensures `address=/<domain>/<lanIP>` is present in dnsmasq,
// replacing any stale entry for the same domain.
func setSplitHorizon(domain, lanIP string) error {
	entry := fmt.Sprintf("/%s/%s", domain, lanIP)
	existing, _ := uci.UciTree.Get("dhcp", uci.DnsmasqSection, "address")
	list := append(filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "/"+domain+"/")
	}), entry)

	// Write as a UCI list: OpenWRT's dnsmasq init reads `address` via
	// config_list_foreach, which ignores the single-value `option` form.
	if ok := uci.UciTree.SetType("dhcp", uci.DnsmasqSection, "address", gouci.TypeList, list...); !ok {
		return fmt.Errorf("set dnsmasq address for %s", domain)
	}
	return nil
}

// setCaptivePortalOption reconciles the RFC 8910 option-114 advertisement on
// the DHCP pool serving ifname: when advertise is true the RFC 8908 API URL is
// present (replacing any prior option-114 entry); when false any option-114
// entry is removed. The pool is resolved via its `interface` option — UCI dhcp
// section names only match interface names by convention, and a pool an admin
// created/renamed by hand must still be found. An interface with no DHCP pool
// (e.g. one not serving DHCP at all) is silently skipped.
func setCaptivePortalOption(domain, ifname string, advertise bool) error {
	section, ok := uci.NewUciDhcpApi().GetSection(ifname)
	if !ok {
		return nil
	}

	existing, _ := uci.UciTree.Get("dhcp", section, "dhcp_option")
	list := filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "114,")
	})
	if advertise {
		list = append(list, fmt.Sprintf("114,https://%s/api/captive", domain))
	}

	if len(list) == 0 {
		// Nothing left in the list — drop the option instead of writing an
		// empty list (which go-uci refuses).
		uci.UciTree.Del("dhcp", section, "dhcp_option")
		return nil
	}

	// Write as a UCI list: dnsmasq's `dhcp_option` is read via
	// config_list_foreach, so the single-value `option` form is ignored.
	if ok := uci.UciTree.SetType("dhcp", section, "dhcp_option", gouci.TypeList, list...); !ok {
		return fmt.Errorf("set dhcp_option 114 on pool %s", section)
	}
	return nil
}

// filterOut returns the elements of list for which drop returns false.
func filterOut(list []string, drop func(string) bool) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		if !drop(v) {
			out = append(out, v)
		}
	}
	return out
}
