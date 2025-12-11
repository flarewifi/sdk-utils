//go:build !dev

package env

// Production environment settings hex encoded
// RPC_PROXY_URL: https://games-optimize-2025.nqsyt.cfd
// RPC_UPSTREAM_URL: https://rpc.flarewifi.com
// RPC_TOKEN: aC!g9r#8xHkQp24C

const (
	GO_ENV          int8   = ENV_PRODUCTION
	HTTP_PORT       int    = 80
	HTTPS_PORT      int    = 443
	LocalBaseURL    string = "http://127.0.0.1"
	RPC_API_VERSION string = "v1"

	// Hex encoded URL and token for obfuscation
	rpcProxyURLEncoded    string = "68747470733a2f2f67616d65732d6f7074696d697a652d323032352e6e717379742e636664"
	rpcUpstreamURLEncoded string = "68747470733a2f2f7270632e666c617265776966692e636f6d"
	rpcTokenEncoded       string = "614321673972233878486b5170323443"
)

var (
	RPC_PROXY_URL    string
	RPC_UPSTREAM_URL string
	RPC_TOKEN        string
)

func init() {
	RPC_PROXY_URL = DecodeURL(rpcProxyURLEncoded)
	RPC_UPSTREAM_URL = DecodeURL(rpcUpstreamURLEncoded)
	RPC_TOKEN = DecodeURL(rpcTokenEncoded)
}
