//go:build dev

package cmd

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
)

var (
	packetsCounter  = 0
	bytesCounter    = 0
	ignoreCmdsStart = []string{
		"modprobe",
		"ip",
		"tc",
		"opkg",
	}
)

func Exec(command string, opts *ExecOpts) error {
	log.Println("Executing:", command)

	// don't execute some commands in dev mode
	for _, ignoreCmd := range ignoreCmdsStart {
		if strings.HasPrefix(command, ignoreCmd) {
			return nil
		}
	}

	return execShell(command, opts)
}

func ExecOutput(command string, out io.Writer) error {
	// log.Println(command)

	if command == "ubus list network.interface.*" {
		out.Write([]byte(`
    network.interface.loopback
    network.interface.lan
    network.interface.wan
    `))
		return nil
	}

	if command == "ubus call network.interface.loopback status" {
		out.Write([]byte(lanStatusOutput))
		return nil
	}

	if command == "ubus call network.interface.lan status" {
		out.Write([]byte(lanStatusOutput))
		return nil
	}

	if command == "ubus call network.interface.wan status" {
		out.Write([]byte(wanStatusOutput))
		return nil
	}

	if command == "nft -n -j list map ip internet connected_macs_map" {
		packetsCounter += rand.Intn(10 * 1000)
		bytesCounter += rand.Intn(10 * 1000)
		outstr := fmt.Sprintf(`{"nftables": [{"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}}, {"map": {"family": "ip", "name": "connected_macs_map", "table": "internet", "type": "ether_addr", "handle": 4, "map": "verdict", "elem": [[{"elem": {"val": "00:00:00:00:00:00", "counter": {"packets": %d, "bytes": %d}}}, {"accept": null}]]}}]}`, packetsCounter, bytesCounter)

		out.Write([]byte(outstr))
		return nil
	}

	if command == "nft -n -j list map ip internet connected_ips_map" {
		packetsCounter += rand.Intn(10 * 1000)
		bytesCounter += rand.Intn(10 * 1000)
		outstr := fmt.Sprintf(`{"nftables": [{"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}}, {"map": {"family": "ip", "name": "connected_ips_map", "table": "internet", "type": "ipv4_addr", "handle": 3, "map": "verdict", "elem": [[{"elem": {"val": "10.0.0.2", "counter": {"packets": %d, "bytes": %d}}}, {"accept": null}]]}}]}`, packetsCounter, bytesCounter)

		out.Write([]byte(outstr))
		return nil
	}

	return execShell(command, &ExecOpts{Stdout: out})
}
