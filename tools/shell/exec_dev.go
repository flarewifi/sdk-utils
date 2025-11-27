//go:build dev

package shell

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"strings"
)

var (
	fakeClients = []struct {
		mac     string
		ip      string
		packets int
		bytes   int
	}{
		{"aa:bb:cc:dd:ee:01", "10.0.0.101", 0, 0},
		{"aa:bb:cc:dd:ee:02", "10.0.0.102", 0, 0},
		{"aa:bb:cc:dd:ee:03", "10.0.0.103", 0, 0},
	}
	ignoreCmdsStart = []string{
		"modprobe",
		"ip",
		"tc",
		"nft",
		"opkg",
		"shutdown",
		"reboot",
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

	if command == "nft -n -j list map inet internet connected_macs_map" {
		// Build elements array with multiple clients
		var elements []string
		for i := range fakeClients {
			fakeClients[i].packets += rand.Intn(100) + 10
			fakeClients[i].bytes += rand.Intn(10000) + 1000
			elem := fmt.Sprintf(`[{"elem": {"val": "%s", "counter": {"packets": %d, "bytes": %d}}}, {"accept": null}]`,
				fakeClients[i].mac, fakeClients[i].packets, fakeClients[i].bytes)
			elements = append(elements, elem)
		}

		outstr := fmt.Sprintf(`{"nftables": [{"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}}, {"map": {"family": "inet", "name": "connected_macs_map", "table": "internet", "type": "ether_addr", "handle": 4, "map": "verdict", "elem": [%s]}}]}`,
			strings.Join(elements, ", "))

		out.Write([]byte(outstr))
		return nil
	}

	if command == "nft -n -j list map inet internet connected_ips_map" {
		// Build elements array with multiple clients
		var elements []string
		for i := range fakeClients {
			elem := fmt.Sprintf(`[{"elem": {"val": "%s", "counter": {"packets": %d, "bytes": %d}}}, {"accept": null}]`,
				fakeClients[i].ip, fakeClients[i].packets, fakeClients[i].bytes)
			elements = append(elements, elem)
		}

		outstr := fmt.Sprintf(`{"nftables": [{"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}}, {"map": {"family": "inet", "name": "connected_ips_map", "table": "internet", "type": "ipv4_addr", "handle": 3, "map": "verdict", "elem": [%s]}}]}`,
			strings.Join(elements, ", "))

		out.Write([]byte(outstr))
		return nil
	}

	return execShell(command, &ExecOpts{Stdout: out})
}

func execShell(command string, opts *ExecOpts) (err error) {
	shells := []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	var shell string
	for _, s := range shells {
		if _, err := exec.LookPath(s); err == nil {
			shell = s
			break
		}
	}
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)

	if opts != nil {
		if opts.Stdout != nil {
			cmd.Stdout = opts.Stdout
		}
		if opts.Stderr != nil {
			cmd.Stderr = opts.Stderr
		}
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if len(opts.Env) > 0 {
			cmd.Env = opts.Env
		}
	}

	var stderr strings.Builder
	if opts == nil || opts.Stderr == nil {
		cmd.Stderr = &stderr
	}

	log.Printf("Executing '%s': %s\n", shell, command)

	if err = cmd.Run(); err != nil {
		if stderr.String() != "" {
			err = errors.New(stderr.String())
		}
	}

	return err
}
