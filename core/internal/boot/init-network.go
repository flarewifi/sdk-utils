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

	return nil
}
