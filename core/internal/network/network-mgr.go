package network

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"core/internal/utils/ubus"
	"tools/config"
	jobque "tools/job-que"
)

const defaultSpeed int = 100 //mbits download/upload per inteface

var lanMap = sync.Map{}
var netQue = jobque.NewJobQue[any]()

func addLan(lan *NetworkLan) {
	lanMap.Store(lan.Name(), lan)
}

func listenLanEvents(lan *NetworkLan) {
	ch := ubus.ListenInterface(lan.Name())
	for evt := range ch {
		netQue.Exec(func() (any, error) {
			if evt.Event == ubus.IfEventDown && lan.Up() {
				lan.SetStatus(false)
			}

			if evt.Event == ubus.IfEventUp && !lan.Up() {
				time.Sleep(1000 * time.Millisecond) // add delay to wait for complete network bootup

				err := lan.ResetTc()
				if err != nil {
					log.Println(err)
					return nil, err
				}

				lan.SetStatus(true)
			}

			return nil, nil
		})

		log.Println("Interface event: ", evt)
	}
}

func SetupLanInterfaces() (err error) {
	ifaces, err := ubus.GetInterfaceNames()
	log.Println("ubus.GetNetworkInterfaces(): ", ifaces)
	if err != nil {
		return err
	}

	for _, ifname := range ifaces {
		cfg, err := config.ReadBandwidthConfig()
		if err != nil {
			return err
		}

		_, ok := cfg.Lans[ifname]
		if ok {
			lan := NewNetworkLan(ifname)
			err := lan.SetupCaptivePortal()
			if err != nil {
				return err
			}

			err = lan.SetupHFSC()
			if err != nil {
				return err
			}
			go listenLanEvents(lan)

			addLan(lan)
		}
	}

	return nil
}

// FindByIp returns the lan instance from lanMap if the given ip is in the subnet of lan ip.
func FindByIp(clientIp string) (*NetworkLan, error) {
	var result *NetworkLan

	log.Println("Finding lan for ip: ", clientIp)
	lanMap.Range(func(key, value any) bool {
		lan := value.(*NetworkLan)
		// get lan subnet net.CIDR
		lanIpV4, err := lan.GetInterface().IpV4Addr()
		if err != nil {
			return true
		}

		_, lanCidr, err := net.ParseCIDR(fmt.Sprintf("%s/%d", lanIpV4.Addr, lanIpV4.Netmask))
		if err != nil {
			return true
		}

		ip := net.ParseIP(clientIp)
		if lanCidr.Contains(ip) {
			result = lan
			return false // stop the iteration
		}

		return true
	})

	if result == nil {
		return nil, errors.New("lan not found")
	}

	return result, nil
}

// FindByName returns the lan instance from lanMap if the given name is in the lanMap.
func FindByName(ifname string) (*NetworkLan, error) {
	lan, ok := lanMap.Load(ifname)
	if !ok {
		return nil, errors.New("lan not found")
	}
	return lan.(*NetworkLan), nil
}

// FindAll returns all lan instances from lanMap.
func FindAll() []*NetworkLan {
	lans := []*NetworkLan{}
	lanMap.Range(func(key, value any) bool {
		lan := value.(*NetworkLan)
		lans = append(lans, lan)
		return true
	})
	return lans
}
