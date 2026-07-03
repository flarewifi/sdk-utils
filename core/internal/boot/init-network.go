package boot

import (
	"fmt"

	"core/internal/modules/nftables"
	"core/internal/network"
	sdkapi "sdk/api"
)

func InitNetwork(logger sdkapi.ILoggerApi) (err error) {
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
	// registered so the main interface can be resolved.
	//
	// Best-effort by design: the firewall base chains + LANs + bandwidth TC are
	// already up at this point, so a portal-reconcile failure must NOT fail
	// InitNetwork — doing so makes the caller skip RunNetworkReadyCallbacks (see
	// boot/init.go), which strands every Network().OnReady() consumer (whitelist,
	// tailscale, coinslot, …) and leaves networkReady=false forever. Log and
	// continue instead.
	if perr := network.ApplyPortalConfig(); perr != nil && logger != nil {
		logger.Error(fmt.Sprintf("boot: portal config reconcile failed (continuing): %v", perr))
	}

	return nil
}
