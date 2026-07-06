//go:build dev

package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"strings"
	"time"
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
		"apk",
		"shutdown",
		"reboot",
		"halt",
		// OpenWRT service management (e.g. service dnsmasq reload, and the
		// underlying /etc/init.d/ scripts) doesn't exist in the dev container;
		// treat it as a no-op.
		"service ",
		"/etc/init.d/",
	}

	// FakeSysupgradeValidationSuccess controls whether sysupgrade -T returns success or failure in dev mode
	// Set to false to test firmware validation failure scenarios
	FakeSysupgradeValidationSuccess = true

	// FakeSysupgradeExecSuccess controls whether the actual sysupgrade (flash+reboot)
	// command succeeds in dev mode. Set to false to test the flash-failure UI.
	FakeSysupgradeExecSuccess = true
)

func Exec(command string, opts *ExecOpts) error {
	// Handle sysupgrade -T command for firmware validation in dev mode
	if strings.HasPrefix(command, "sysupgrade -T") {
		time.Sleep(5 * time.Second)
		if FakeSysupgradeValidationSuccess {
			return nil
		}
		return errors.New("firmware validation failed: incompatible firmware")
	}

	// Handle other sysupgrade commands (actual upgrade) - ignore in dev mode
	if strings.HasPrefix(command, "sysupgrade") {
		time.Sleep(5 * time.Second)
		if FakeSysupgradeExecSuccess {
			return nil
		}
		return errors.New("sysupgrade: flash failed: short write to mtd device")
	}

	// don't execute some commands in dev mode
	for _, ignoreCmd := range ignoreCmdsStart {
		if strings.HasPrefix(command, ignoreCmd) {
			return nil
		}
	}

	return execShell(command, opts)
}

func ExecOutput(command string, out io.Writer) error {
	// Neither opkg nor apk is available in the dev container; treat package
	// queries as returning no output (nothing installed) so plugin
	// system_packages installs become a no-op in dev.
	if strings.HasPrefix(command, "opkg") || strings.HasPrefix(command, "apk") {
		return nil
	}

	// Handle nftables firewall commands
	if strings.Contains(command, "nft -j list table inet internet") {
		out.Write([]byte(getFakeNFTTableJSON()))
		return nil
	}

	if strings.Contains(command, "nft -a list chain inet internet open_ip_prerouting") {
		out.Write([]byte(getFakeOpenIpPrerouting()))
		return nil
	}

	if strings.Contains(command, "nft -a list chain inet internet open_ip_forward") {
		out.Write([]byte(getFakeOpenIpForward()))
		return nil
	}

	if strings.Contains(command, "nft -a list chain inet internet prerouting") {
		out.Write([]byte(getFakePreroutingChain()))
		return nil
	}

	if strings.Contains(command, "nft -j -a list chain inet internet forward") {
		out.Write([]byte(getFakeForwardChainJSON()))
		return nil
	}

	if strings.Contains(command, "nft -a list chain inet internet forward") {
		out.Write([]byte(getFakeForwardChain()))
		return nil
	}

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

	if err = cmd.Run(); err != nil {
		if stderr.String() != "" {
			err = errors.New(stderr.String())
		}
	}

	return err
}

// ExecWithContext executes shell command with context cancellation support (dev version)
func ExecWithContext(ctx context.Context, command string, opts *ExecOpts) error {
	// don't execute some commands in dev mode
	for _, ignoreCmd := range ignoreCmdsStart {
		if strings.HasPrefix(command, ignoreCmd) {
			return nil
		}
	}

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

	cmd := exec.CommandContext(ctx, shell, "-c", command)

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

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out: %w", err)
		}
		if stderr.String() != "" {
			err = errors.New(stderr.String())
		}
		return err
	}

	return nil
}

// getFakeNFTTableJSON returns a fake JSON response for nft table listing
func getFakeNFTTableJSON() string {
	return `{
  "nftables": [
    {"metainfo": {"version": "1.0.0", "release_name": "Fearless Fosdick #3", "json_schema_version": 1}},
    {"table": {"family": "inet", "name": "internet", "handle": 1}},
    {"chain": {"family": "inet", "table": "internet", "name": "prerouting", "handle": 1, "type": "nat", "hook": "prerouting", "prio": -100, "policy": "accept"}},
    {"chain": {"family": "inet", "table": "internet", "name": "forward", "handle": 2, "type": "filter", "hook": "forward", "prio": 0, "policy": "accept"}},
    {"chain": {"family": "inet", "table": "internet", "name": "open_ip_prerouting", "handle": 10}},
    {"chain": {"family": "inet", "table": "internet", "name": "open_ip_forward", "handle": 11}}
  ]
}`
}

// getFakeOpenIpPrerouting returns a fake response for open_ip_prerouting chain listing
func getFakeOpenIpPrerouting() string {
	return `table inet internet {
	chain open_ip_prerouting {
	}
}`
}

// getFakeOpenIpForward returns a fake response for open_ip_forward chain listing
func getFakeOpenIpForward() string {
	return `table inet internet {
	chain open_ip_forward {
	}
}`
}

// getFakePreroutingChain returns a fake response for prerouting chain listing
func getFakePreroutingChain() string {
	return `table inet internet {
	chain prerouting {
		type nat hook prerouting priority -100; policy accept;
		counter packets 0 bytes 0 jump open_ip_prerouting # handle 100
	}
}`
}

// getFakeForwardChain returns a fake response for forward chain listing (text format)
func getFakeForwardChain() string {
	return `table inet internet {
	chain forward {
		type filter hook forward priority 0; policy accept;
		counter packets 0 bytes 0 jump open_ip_forward # handle 200
	}
}`
}

// getFakeForwardChainJSON returns a fake JSON response for forward chain listing
// This includes the core firewall rules (ct state invalid, IP/MAC vmaps, etc.)
func getFakeForwardChainJSON() string {
	return `{
  "nftables": [
    {"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}},
    {"rule": {
      "family": "inet",
      "table": "internet",
      "chain": "forward",
      "handle": 1,
      "expr": [
        {"match": {"op": "==", "left": {"ct": {"key": "state"}}, "right": "invalid"}},
        {"counter": {"packets": 0, "bytes": 0}},
        {"drop": null}
      ]
    }},
    {"rule": {
      "family": "inet",
      "table": "internet",
      "chain": "forward",
      "handle": 2,
      "expr": [
        {"vmap": {
          "key": {"payload": {"protocol": "ip", "field": "daddr"}},
          "data": "@connected_ips_map"
        }}
      ]
    }},
    {"rule": {
      "family": "inet",
      "table": "internet",
      "chain": "forward",
      "handle": 3,
      "expr": [
        {"vmap": {
          "key": {"payload": {"protocol": "ip6", "field": "daddr"}},
          "data": "@connected_ips6_map"
        }}
      ]
    }},
    {"rule": {
      "family": "inet",
      "table": "internet",
      "chain": "forward",
      "handle": 4,
      "expr": [
        {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": ["established", "related"]}},
        {"counter": {"packets": 0, "bytes": 0}},
        {"accept": null}
      ]
    }},
    {"rule": {
      "family": "inet",
      "table": "internet",
      "chain": "forward",
      "handle": 5,
      "expr": [
        {"vmap": {
          "key": {"payload": {"protocol": "ether", "field": "saddr"}},
          "data": "@connected_macs_map"
        }}
      ]
    }}
  ]
}`
}
