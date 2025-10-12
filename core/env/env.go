//go:build !dev && !staging

package env

const (
	GO_ENV          int8   = ENV_PRODUCTION
	HTTP_PORT       int    = 80
	RPC_TOKEN              = "aC!g9r#8xHkQp24C"
	RPC_API_VERSION string = "v1"
	RPC_BASE_URL    string = "https://rpc-core.nqsyt.cfd"
)

var (
	BuildTags string = "prod"
)
