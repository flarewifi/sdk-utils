package boot

import (
	"log"

	"core/internal/modules/nftables"
	"core/internal/modules/ubus"
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

	ubus.Listen()

	return nil
}
