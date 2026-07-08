package api

import (
	"errors"

	"core/internal/modules/ubus"
	cnet "core/internal/network"
	"core/utils/config"
	sdkapi "sdk/api"
)

func NewInterfaceCfgApi() *InterfaceCfgApi {
	return &InterfaceCfgApi{}
}

type InterfaceCfgApi struct{}

// Get returns the EFFECTIVE per-interface captive-portal state, not just the
// raw persisted document: an interface with no explicit entry in
// interfaces.json still resolves via cnet.IsCaptivePortalEnabled's
// primary-LAN-bridge default (the same fallback that lets a fresh,
// never-configured machine work out of the box). This keeps that default
// defined in exactly one place — callers get a correct answer from a plain
// map lookup instead of re-deriving the device-based fallback themselves.
func (self *InterfaceCfgApi) Get() (sdkapi.InterfaceCfg, error) {
	cfg, err := config.ReadInterfacesConfig()
	if err != nil {
		return sdkapi.InterfaceCfg{}, err
	}

	ifaces, err := ubus.GetNetworkInterfaces()
	if err != nil {
		return sdkapi.InterfaceCfg{}, err
	}

	lans := make(map[string]sdkapi.LanInterfaceCfg, len(ifaces))
	for ifname := range ifaces {
		persisted := cfg.LanInterfaces[ifname]
		lans[ifname] = sdkapi.LanInterfaceCfg{
			EnableCaptivePortal: cnet.IsCaptivePortalEnabled(ifname),
			IpAddress:           persisted.IpAddress,
			Netmask:             persisted.Netmask,
		}
	}

	return sdkapi.InterfaceCfg{
		PortalInterface: cfg.PortalInterface,
		LanInterfaces:   lans,
	}, nil
}

func (self *InterfaceCfgApi) Save(cfg sdkapi.InterfaceCfg) error {
	internalCfg := toInternalInterfacesCfg(cfg)

	captiveCount := 0
	for _, lan := range internalCfg.LanInterfaces {
		if lan.EnableCaptivePortal {
			captiveCount++
		}
	}

	if captiveCount > 0 {
		lan, ok := internalCfg.LanInterfaces[internalCfg.PortalInterface]
		if !ok || !lan.EnableCaptivePortal {
			return errors.New("portal interface must have captive portal enabled")
		}
	} else {
		internalCfg.PortalInterface = ""
	}

	if err := config.WriteInterfacesConfig(internalCfg); err != nil {
		return err
	}

	return cnet.ReconcileInterfaces()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func toInternalInterfacesCfg(cfg sdkapi.InterfaceCfg) config.InterfacesCfg {
	lans := make(map[string]config.LanInterfaceCfg, len(cfg.LanInterfaces))
	for ifname, lan := range cfg.LanInterfaces {
		lans[ifname] = config.LanInterfaceCfg{
			EnableCaptivePortal: lan.EnableCaptivePortal,
			IpAddress:           lan.IpAddress,
			Netmask:             lan.Netmask,
		}
	}
	return config.InterfacesCfg{
		PortalInterface: cfg.PortalInterface,
		LanInterfaces:   lans,
	}
}
