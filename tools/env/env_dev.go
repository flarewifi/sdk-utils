//go:build dev

package env

const (
	GO_ENV          int8   = ENV_DEV
	HTTP_PORT       int    = 3000
	LocalBaseURL    string = "http://localhost:3000"
	RPC_TOKEN       string = "xxxxxxxxxx"
	RPC_API_VERSION string = "v1"
	RPC_BASE_URL    string = "http://flarehotspot.rpc-dev.com"
)
