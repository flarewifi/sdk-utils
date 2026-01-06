package network

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"core/internal/modules/ubus"
	"core/utils/config"
	jobque "core/utils/job-que"
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
				log.Printf("LAN interface '%s' went DOWN", lan.Name())
				lan.SetStatus(false)
			}

			if evt.Event == ubus.IfEventUp && !lan.Up() {
				log.Printf("LAN interface '%s' came UP, reinitializing...", lan.Name())
				time.Sleep(1000 * time.Millisecond) // add delay to wait for complete network bootup

				// Reinitialize TC (handles IP changes and ensures proper setup)
				err := lan.ReinitializeTc()
				if err != nil {
					log.Printf("ERROR: Failed to reinitialize TC for LAN '%s': %v", lan.Name(), err)
					return nil, err
				}

				lan.SetStatus(true)
				log.Printf("LAN interface '%s' reinitialized successfully", lan.Name())
			}

			return nil, nil
		})

		log.Println("Interface event: ", evt)
	}
}

func SetupLanInterfaces() (err error) {
	log.Println("SetupLanInterfaces: Starting LAN interface setup...")

	ifaces, err := ubus.GetInterfaceNames()
	log.Println("ubus.GetNetworkInterfaces(): ", ifaces)
	if err != nil {
		log.Printf("ERROR: Failed to get interface names from UBUS: %v", err)
		return err
	}

	cfg, err := config.ReadBandwidthConfig()
	if err != nil {
		log.Printf("ERROR: Failed to read bandwidth config: %v", err)
		return err
	}
	log.Printf("Bandwidth config loaded. Configured LANs: %v", getConfiguredLanNames(cfg))

	lanCount := 0
	for _, ifname := range ifaces {
		_, ok := cfg.Lans[ifname]
		if ok {
			lan := NewNetworkLan(ifname)

			err = lan.SetupHFSC()
			if err != nil {
				log.Printf("ERROR: Failed to setup HFSC for interface %s: %v", ifname, err)
				return err
			}
			go listenLanEvents(lan)

			addLan(lan)
			lanCount++
			log.Printf("LAN interface '%s' added to lanMap", ifname)
		} else {
			log.Printf("Interface '%s' not found in bandwidth config, skipping", ifname)
		}
	}

	log.Printf("SetupLanInterfaces complete: %d LAN(s) configured", lanCount)

	if lanCount == 0 {
		log.Println("WARNING: No LAN interfaces were configured! Check bandwidth.json config.")
	}

	return nil
}

func getConfiguredLanNames(cfg config.BandwdCfg) []string {
	names := []string{}
	for name := range cfg.Lans {
		names = append(names, name)
	}
	return names
}

// FindByIp returns the lan instance from lanMap if the given ip is in the subnet of lan ip.
func FindByIp(clientIp string) (*NetworkLan, error) {
	var result *NetworkLan
	var lastError error
	var checkedLans []string

	lanCount := GetLanCount()
	log.Printf("Finding LAN for client IP: %s (total LANs in map: %d)", clientIp, lanCount)

	lanMap.Range(func(key, value any) bool {
		lan := value.(*NetworkLan)
		lanName := lan.Name()

		// get lan subnet net.CIDR
		lanIpV4, err := lan.GetInterface().IpV4Addr()
		if err != nil {
			log.Printf("Failed to get IPv4 address for LAN '%s': %v", lanName, err)
			lastError = fmt.Errorf("LAN '%s': %w", lanName, err)
			checkedLans = append(checkedLans, fmt.Sprintf("%s (error: %v)", lanName, err))
			return true
		}

		cidrStr := fmt.Sprintf("%s/%d", lanIpV4.Addr, lanIpV4.Netmask)
		_, lanCidr, err := net.ParseCIDR(cidrStr)
		if err != nil {
			log.Printf("Failed to parse CIDR '%s' for LAN '%s': %v", cidrStr, lanName, err)
			lastError = fmt.Errorf("LAN '%s': invalid CIDR %s: %w", lanName, cidrStr, err)
			checkedLans = append(checkedLans, fmt.Sprintf("%s (invalid CIDR: %s)", lanName, cidrStr))
			return true
		}

		ip := net.ParseIP(clientIp)
		if ip == nil {
			log.Printf("Invalid client IP address: %s", clientIp)
			lastError = fmt.Errorf("invalid client IP: %s", clientIp)
			return false
		}

		log.Printf("Checking if %s is in %s (LAN: %s)", clientIp, cidrStr, lanName)
		checkedLans = append(checkedLans, fmt.Sprintf("%s (%s)", lanName, cidrStr))

		if lanCidr.Contains(ip) {
			log.Printf("Client IP %s matched LAN '%s' (%s)", clientIp, lanName, cidrStr)
			result = lan
			return false // stop the iteration
		}

		return true
	})

	if result == nil {
		log.Printf("No matching LAN found for IP %s. Checked LANs: %v", clientIp, checkedLans)
		if lastError != nil {
			return nil, fmt.Errorf("no matching LAN found for IP %s: %w", clientIp, lastError)
		}
		return nil, fmt.Errorf("no matching LAN found for IP %s (checked: %v)", clientIp, checkedLans)
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

// GetLanCount returns the number of LANs in lanMap (for debugging)
func GetLanCount() int {
	count := 0
	lanMap.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
