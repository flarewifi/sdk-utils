package cfgapi

import (
	"core/internal/config"
	sdkapi "sdk/api"
)

func NewBandwdCfgApi() *BandwdCfgApi {
	return &BandwdCfgApi{}
}

type BandwdCfgApi struct{}

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

	oldCfg.Lans[ifname] = config.IfCfg{
		UseGlobal:       cfg.UseGlobal,
		GlobalDownMbits: cfg.GlobalDownMbits,
		GlobalUpMbits:   cfg.GlobalUpMbits,
		UserDownMbits:   cfg.UserDownMbits,
		UserUpMbits:     cfg.UserUpMbits,
	}

	return config.WriteBandwidthConfig(oldCfg)
}
