//go:build dev

package env

const (
	GO_ENV        int8   = ENV_DEV
	HTTP_PORT     int    = 3000
	HTTPS_PORT    int    = 443
	LocalBaseURL  string = "http://localhost:3000"
	RPC_TOKEN     string = "xxxxxxxxxx"
	RPC_PROXY_URL string = "http://cf-proxy.flare-local.com"
	SERVER_DOMAIN string = "flare-local.com"
)
