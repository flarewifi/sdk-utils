// Package captivedns writes the dnsmasq configuration that makes the shared
// captive-portal hostname resolve locally (split-horizon DNS) and advertises the
// RFC 8908 Captive Portal API to clients via the RFC 8910 DHCP option 114.
package captivedns

import (
	"fmt"
	"strings"

	gouci "github.com/digineo/go-uci"

	"core/internal/modules/uci"
	"core/utils/config"
	cmd "core/utils/shell"
)

const (
	// dnsmasqSection is the UCI section that holds dnsmasq's `address` list.
	// The dnsmasq section in /etc/config/dhcp is anonymous, so it must be
	// addressed by its unnamed selector (@dnsmasq[0]); a lookup by the literal
	// name "dnsmasq" never matches and makes every Set on it fail.
	dnsmasqSection = "@dnsmasq[0]"
	// lanSection is the UCI dhcp pool whose `dhcp_option` list we extend.
	lanSection = "lan"
)

// Setup points <custom_domain> at the router's LAN IP for connected clients
// (split-horizon) and advertises the captive-portal API URL via DHCP option 114,
// then reloads dnsmasq. It is idempotent: prior entries for the same domain and
// for option 114 are replaced rather than duplicated. A missing custom_domain or
// LAN IP is a no-op (nothing to advertise).
func Setup(lanIP string) error {
	cfg, err := config.GetCachedAppConfig()
	if err != nil {
		return fmt.Errorf("read app config: %w", err)
	}

	// A custom_domain is the cloud-issued portal hostname. When it is empty there
	// is no portal host to advertise, so split-horizon DNS and the DHCP option-114
	// portal URL are skipped (the no-op below). Holds in dev, staging, and prod.
	domain := strings.TrimSpace(cfg.CustomDomain)
	if domain == "" || lanIP == "" {
		return nil
	}

	if err := setSplitHorizon(domain, lanIP); err != nil {
		return err
	}
	if err := setCaptivePortalOption(domain); err != nil {
		return err
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
	existing, _ := uci.UciTree.Get("dhcp", dnsmasqSection, "address")
	list := append(filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "/"+domain+"/")
	}), entry)

	// Write as a UCI list: OpenWRT's dnsmasq init reads `address` via
	// config_list_foreach, which ignores the single-value `option` form.
	if ok := uci.UciTree.SetType("dhcp", dnsmasqSection, "address", gouci.TypeList, list...); !ok {
		return fmt.Errorf("set dnsmasq address for %s", domain)
	}
	return nil
}

// setCaptivePortalOption ensures the RFC 8910 option 114 advertises the RFC 8908
// API URL, replacing any prior option-114 entry.
func setCaptivePortalOption(domain string) error {
	value := fmt.Sprintf("114,https://%s/api/captive", domain)
	existing, _ := uci.UciTree.Get("dhcp", lanSection, "dhcp_option")
	list := append(filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "114,")
	}), value)

	// Write as a UCI list: dnsmasq's `dhcp_option` is read via
	// config_list_foreach, so the single-value `option` form is ignored.
	if ok := uci.UciTree.SetType("dhcp", lanSection, "dhcp_option", gouci.TypeList, list...); !ok {
		return fmt.Errorf("set dhcp_option 114")
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
