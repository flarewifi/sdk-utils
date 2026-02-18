package api

import (
	"context"

	"core/internal/sessmgr"
	"core/utils/config"
	sdkapi "sdk/api"
)

func NewBandwdCfgApi(sessionMgr *sessmgr.SessionsMgr) *BandwdCfgApi {
	return &BandwdCfgApi{sessionMgr: sessionMgr}
}

type BandwdCfgApi struct {
	sessionMgr *sessmgr.SessionsMgr
}

func (c *BandwdCfgApi) Get(ifname string) (sdkapi.IBandwdCfg, bool) {
	cfg, err := config.ReadBandwidthConfig()
	if err != nil {
		return sdkapi.IBandwdCfg{}, false
	}

	bcfg, ok := cfg.Lans[ifname]
	if !ok {
		return sdkapi.IBandwdCfg{}, false
	}

	return sdkapi.IBandwdCfg{
		UseGlobal:       bcfg.UseGlobal,
		GlobalDownMbits: bcfg.GlobalDownMbits,
		GlobalUpMbits:   bcfg.GlobalUpMbits,
		UserDownMbits:   bcfg.UserDownMbits,
		UserUpMbits:     bcfg.UserUpMbits,
	}, true
}

func (c *BandwdCfgApi) Save(ifname string, cfg sdkapi.IBandwdCfg) error {
	oldCfg, err := config.ReadBandwidthConfig()
	if err != nil {
		return err
	}

	// Initialize the map if it's nil to prevent panic
	if oldCfg.Lans == nil {
		oldCfg.Lans = make(map[string]config.IfCfg)
	}

	oldCfg.Lans[ifname] = config.IfCfg{
		UseGlobal:       cfg.UseGlobal,
		GlobalDownMbits: cfg.GlobalDownMbits,
		GlobalUpMbits:   cfg.GlobalUpMbits,
		UserDownMbits:   cfg.UserDownMbits,
		UserUpMbits:     cfg.UserUpMbits,
	}

	if err := config.WriteBandwidthConfig(oldCfg); err != nil {
		return err
	}

	// Update running sessions on this interface with the new bandwidth settings
	if c.sessionMgr != nil {
		c.sessionMgr.UpdateInterfaceBandwidth(context.Background(), ifname, cfg)
	}

	return nil
}
