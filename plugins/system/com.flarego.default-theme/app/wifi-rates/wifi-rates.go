package wifirates

import (
	"encoding/json"
	"os"
	"path/filepath"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type PauseSessionSetting struct {
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type PauseSessionConfig struct {
	Unlimited bool                  `json:"unlimited"`
	Settings  []PauseSessionSetting `json:"settings"`
}

func GetPauseConfig(api sdkapi.IPluginApi) PauseSessionConfig {
	filePath := filepath.Join(sdkutils.PathConfigDir, "plugins", "com.flarego.wifi-hotspot", "session_settings")
	file, err := os.ReadFile(filePath)
	if err != nil {
		return PauseSessionConfig{
			Settings: []PauseSessionSetting{
				{Amount: 1, Count: 0},
				{Amount: 5, Count: 3},
				{Amount: 10, Count: 6},
				{Amount: 20, Count: 12},
			},
		}
	}
	var cfg PauseSessionConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		return PauseSessionConfig{}
	}
	return cfg
}

// PauseCountFor returns the pause count for a given payment amount.
// If unlimited, returns -1.
func (c PauseSessionConfig) PauseCountFor(amount float64) int {
	if c.Unlimited {
		return -1
	}
	for _, s := range c.Settings {
		if s.Amount == amount {
			return s.Count
		}
	}
	return 0
}

type PaymentSetting struct {
	Amount         float64 `json:"amount"`
	DataMb         int     `json:"data_mb"`
	TimeMins       int     `json:"time_mins"`
	DataCapEnabled bool    `json:"data_cap_enabled"`
	ExpiryEnabled  bool    `json:"expiry_enabled"`
	ExpiryTime     int     `json:"expiry_time"`
	ExpiryUnit     string  `json:"expiry_unit"`
}

type PaymentSettings []PaymentSetting

var DefaultPaymentSettings = PaymentSettings{
	{
		Amount:         1,
		DataMb:         10,
		TimeMins:       15,
		DataCapEnabled: false,
		ExpiryEnabled:  false,
		ExpiryTime:     0,
		ExpiryUnit:     "hours",
	},
	{
		Amount:         5,
		DataMb:         50,
		TimeMins:       60,
		DataCapEnabled: false,
		ExpiryEnabled:  false,
		ExpiryTime:     0,
		ExpiryUnit:     "hours",
	},
	{
		Amount:         10,
		DataMb:         100,
		TimeMins:       180,
		DataCapEnabled: false,
		ExpiryEnabled:  true,
		ExpiryTime:     3,
		ExpiryUnit:     "days",
	},
	{
		Amount:         20,
		DataMb:         100,
		TimeMins:       180,
		DataCapEnabled: false,
		ExpiryEnabled:  true,
		ExpiryTime:     7,
		ExpiryUnit:     "days",
	},
}

func GetPaymentConfig(api sdkapi.IPluginApi) PaymentSettings {
	var settings PaymentSettings

	filePath := filepath.Join(sdkutils.PathConfigDir, "plugins", "com.flarego.wifi-hotspot", "payment_settings")
	file, err := os.ReadFile(filePath)

	if err != nil {
		return DefaultPaymentSettings
	}

	if err := json.Unmarshal(file, &settings); err != nil {
		return DefaultPaymentSettings
	}

	return settings
}
