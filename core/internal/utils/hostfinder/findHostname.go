package hostfinder

import (
	"bufio"
	"os"
	"strings"
)

func FindHostname(mac string) (hostname string, err error) {
	leasesFile := "/tmp/dhcp.leases"
	file, err := os.Open(leasesFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K
	// const maxCapacity = 512*1024
	// buf := make([]byte, maxCapacity)
	// scanner.Buffer(buf, maxCapacity)
	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		m := line[1]
		if m == mac {
			h := line[3]
			return h, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
