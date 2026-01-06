//go:build dev

package shell

const lanStatusOutput = `
{
        "up": true,
        "pending": false,
        "available": true,
        "autostart": true,
        "dynamic": false,
        "uptime": 11461,
        "l3_device": "br-lan",
        "proto": "static",
        "device": "br-lan",
        "updated": [
                "addresses"
        ],
        "metric": 0,
        "dns_metric": 0,
        "delegation": true,
        "ipv4-address": [
                {
                        "address": "10.0.0.1",
                        "mask": 20
                }
        ],
        "ipv6-address": [

        ],
        "ipv6-prefix": [

        ],
        "ipv6-prefix-assignment": [
                {
                        "address": "fd49:9c61:f1c0::",
                        "mask": 60,
                        "local-address": {
                                "address": "fd49:9c61:f1c0::1",
                                "mask": 60
                        }
                }
        ],
        "route": [

        ],
        "dns-server": [

        ],
        "dns-search": [

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

        }
}

`
