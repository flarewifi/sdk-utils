package boot

import (
	"log"

	"core/internal/modules/nftables"
	"core/internal/network"
)

func InitNetwork() (err error) {
	err = nftables.Setup()
	if err != nil {
		log.Println(err)
		return err
	}

	err = network.SetupLanInterfaces()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
