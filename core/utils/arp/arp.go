// source: https://github.com/mostlygeek/arp/pull/5/files
package arp

import (
	"bufio"
	"os"
	"strings"
)

type ArpTable map[string]string

const (
	f_IPAddr int = iota
	f_HWType
	f_Flags
	f_HWAddr
)

func Table() ArpTable {
	f, err := os.Open("/proc/net/arp")

	if err != nil {
		return nil
	}

	defer f.Close()

	s := bufio.NewScanner(f)
	s.Scan()

	var table = make(ArpTable)

	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		table[fields[f_IPAddr]] = padMacString(fields[f_HWAddr])
	}

	return table
}

func Search(ip string) (mac string, ok bool) {
	table := Table()
	mac, ok = table[ip]
	return mac, ok
}

// FindIpByMac performs a reverse ARP lookup: given a MAC address, returns the
// first matching IPv4 address found in the ARP table, or empty string if not found.
// The mac parameter is compared case-insensitively after padding.
func FindIpByMac(mac string) string {
	normalized := strings.ToLower(padMacString(mac))
	table := Table()
	for ip, m := range table {
		if strings.ToLower(m) == normalized {
			return ip
		}
	}
	return ""
}
