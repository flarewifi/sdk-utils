package config

const bandwidthJsonFile = "bandwidth.json"

type BandwdCfg struct {
	Lans map[string]IfCfg `json:"lans"`
}

type IfCfg struct {
	UseGlobal       bool `json:"use_global"`
	GlobalDownMbits int  `json:"global_down_mbits"`
	GlobalUpMbits   int  `json:"global_up_mbits"`
	UserDownMbits   int  `json:"user_down_mbits"`
	UserUpMbits     int  `json:"user_up_mbits"`
}

// DefaultIfCfg is the effective bandwidth config for an interface with no explicit
// entry: use the global (LAN-wide) bandwidth with no per-user cap. Global speeds
// are left 0 so the traffic-control layer auto-detects the link speed.
func DefaultIfCfg() IfCfg {
	return IfCfg{UseGlobal: true}
}

// LanCfg returns the bandwidth config for ifname, falling back to DefaultIfCfg()
// (use global bandwidth) when there is no explicit entry. Use this instead of a
// bare cfg.Lans[ifname] so a missing entry defaults to global rather than the
// zero value (which would read as per-user with a 0 cap).
func (cfg BandwdCfg) LanCfg(ifname string) IfCfg {
	if c, ok := cfg.Lans[ifname]; ok {
		return c
	}
	return DefaultIfCfg()
}

func ReadBandwidthConfig() (BandwdCfg, error) {
	var cfg BandwdCfg
	err := readConfigFile(bandwidthJsonFile, &cfg)
	return cfg, err
}

func WriteBandwidthConfig(cfg BandwdCfg) error {
	return writeConfigFile(bandwidthJsonFile, cfg)
}
