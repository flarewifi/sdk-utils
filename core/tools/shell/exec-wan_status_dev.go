//go:build dev

package shell

const wanStatusOutput = `
{
        "up": true,
        "pending": false,
        "available": true,
        "autostart": true,
        "dynamic": false,
        "uptime": 11721,
        "l3_device": "eth1",
        "proto": "dhcp",
        "device": "eth1",
        "metric": 0,
        "dns_metric": 0,
        "delegation": true,
        "ipv4-address": [
                {
                        "address": "192.168.222.25",
                        "mask": 24
                }
        ],
        "ipv6-address": [

        ],
        "ipv6-prefix": [

        ],
        "ipv6-prefix-assignment": [

        ],
        "route": [
                {
                        "target": "0.0.0.0",
                        "mask": 0,
                        "nexthop": "192.168.222.1",
                        "source": "192.168.222.25/32"
                }
        ],
        "dns-server": [
                "192.168.222.1",
                "8.8.8.8"
        ],
        "dns-search": [
                "adohome.net"
        ],
        "neighbors": [

        ],
        "inactive": {
                "ipv4-address": [

                ],
                "ipv6-address": [

                ],
                "route": [

                ],
                "dns-server": [

                ],
                "dns-search": [

                ],
                "neighbors": [

                ]
        },
        "data": {
                "dhcpserver": "192.168.222.1",
                "leasetime": 10080
        }
}`
