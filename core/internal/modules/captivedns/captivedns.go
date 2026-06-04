// Package captivedns writes the dnsmasq configuration that makes the shared
// captive-portal hostname resolve locally (split-horizon DNS) and advertises the
// RFC 8908 Captive Portal API to clients via the RFC 8910 DHCP option 114.
package captivedns

import (
	"fmt"
	"log"
	"strings"

	gouci "github.com/digineo/go-uci"

	"core/internal/modules/uci"
	"core/utils/config"
	"core/utils/env"
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
	log.Printf("captivedns: Setup called with lanIP=%q (GO_ENV=%d, ENV_PRODUCTION=%d)", lanIP, env.GO_ENV, env.ENV_PRODUCTION)

	cfg, err := config.GetCachedAppConfig()
	if err != nil {
		log.Printf("captivedns: read app config FAILED: %v", err)
		return fmt.Errorf("read app config: %w", err)
	}
	log.Printf("captivedns: app config CustomDomain=%q", cfg.CustomDomain)

	if env.GO_ENV == env.ENV_DEV {
		cfg.CustomDomain = "captive.flare-local.com"
	}

	domain := strings.TrimSpace(cfg.CustomDomain)
	if domain == "" || lanIP == "" {
		log.Printf("captivedns: no-op (nothing to advertise): domain=%q lanIP=%q", domain, lanIP)
		return nil
	}
	log.Printf("captivedns: proceeding with domain=%q lanIP=%q", domain, lanIP)

	if err := setSplitHorizon(domain, lanIP); err != nil {
		log.Printf("captivedns: setSplitHorizon FAILED: %v", err)
		return err
	}
	if err := setCaptivePortalOption(domain); err != nil {
		log.Printf("captivedns: setCaptivePortalOption FAILED: %v", err)
		return err
	}

	if err := uci.UciTree.Commit(); err != nil {
		log.Printf("captivedns: uci commit dhcp FAILED: %v", err)
		return fmt.Errorf("uci commit dhcp: %w", err)
	}
	log.Printf("captivedns: uci commit dhcp OK")

	// Reload (not restart) so existing DHCP leases are preserved.
	if err := cmd.Exec("service dnsmasq reload", nil); err != nil {
		log.Printf("captivedns: dnsmasq reload FAILED: %v", err)
		return fmt.Errorf("dnsmasq reload: %w", err)
	}
	log.Printf("captivedns: Setup complete (dnsmasq reloaded)")
	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// setSplitHorizon ensures `address=/<domain>/<lanIP>` is present in dnsmasq,
// replacing any stale entry for the same domain.
func setSplitHorizon(domain, lanIP string) error {
	entry := fmt.Sprintf("/%s/%s", domain, lanIP)
	existing, getOk := uci.UciTree.Get("dhcp", dnsmasqSection, "address")
	log.Printf("captivedns: get dhcp.%s.address ok=%v existing=%v", dnsmasqSection, getOk, existing)
	list := append(filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "/"+domain+"/")
	}), entry)
	log.Printf("captivedns: setting dhcp.%s.address=%v", dnsmasqSection, list)

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
	existing, getOk := uci.UciTree.Get("dhcp", lanSection, "dhcp_option")
	log.Printf("captivedns: get dhcp.%s.dhcp_option ok=%v existing=%v", lanSection, getOk, existing)
	list := append(filterOut(existing, func(v string) bool {
		return strings.HasPrefix(v, "114,")
	}), value)
	log.Printf("captivedns: setting dhcp.%s.dhcp_option=%v", lanSection, list)

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
