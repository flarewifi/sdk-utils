package boot

import (
	"core/internal/modules/nftables"
	"core/internal/network"
)

func InitNetwork() (err error) {
	err = nftables.Setup()
	if err != nil {
		return err
	}

	err = network.SetupLanInterfaces()
	if err != nil {
		return err
	}

	// Apply per-interface portal roles (captive vs open) and point the captive
	// DNAT + split-horizon DNS at the main LAN's IP. Runs after all LANs are
	// registered so the main interface can be resolved. Best-effort: a portal
	// reconcile failure must not block boot (the firewall + LANs are already up).
	if err := network.ApplyPortalConfig(); err != nil {
		return err
	}

	return nil
}
