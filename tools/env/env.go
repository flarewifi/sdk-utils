//go:build !dev

package env

const (
	GO_ENV          int8   = ENV_PRODUCTION
	HTTP_PORT       int    = 80
	HTTPS_PORT      int    = 443
	LocalBaseURL    string = "http://127.0.0.1"
	RPC_API_VERSION string = "v1"

	// Hex encoded URL and token for obfuscation
	rpcBaseURLEncoded string = "68747470733a2f2f67616d65732d636f6e6e6563742e6e717379742e636664"
	rpcTokenEncoded   string = "614321673972233878486b5170323443"
)

var (
	RPC_BASE_URL string
	RPC_TOKEN    string
)

func init() {
	RPC_BASE_URL = DecodeURL(rpcBaseURLEncoded)
	RPC_TOKEN = DecodeURL(rpcTokenEncoded)
}
