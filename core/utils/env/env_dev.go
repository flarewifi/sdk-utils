//go:build dev

package env

const (
	GO_ENV           int8   = ENV_DEV
	HTTP_PORT        int    = 3000
	HTTPS_PORT       int    = 3443
	LocalBaseURL     string = "http://localhost:3000"
	RPC_TOKEN        string = "xxxxxxxxxx"
	RPC_API_VERSION  string = "v1"
	RPC_PROXY_URL    string = "http://cf-proxy.flare-local.com"
	RPC_UPSTREAM_URL string = "http://rpc.flare-local.com"
)
