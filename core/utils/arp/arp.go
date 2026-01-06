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
