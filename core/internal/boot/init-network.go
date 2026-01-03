package boot

import (
	"fmt"
	"log"

	"core/internal/modules/nftables"
	"core/internal/modules/ubus"
	"core/internal/network"
)

func InitNetwork() (err error) {
	fmt.Println("Initializing network...")

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
