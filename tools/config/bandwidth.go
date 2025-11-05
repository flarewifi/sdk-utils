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

func ReadBandwidthConfig() (BandwdCfg, error) {
	var cfg BandwdCfg
	err := readConfigFile(bandwidthJsonFile, &cfg)
	return cfg, err
}

func WriteBandwidthConfig(cfg BandwdCfg) error {
	return writeConfigFile(bandwidthJsonFile, cfg)
}
