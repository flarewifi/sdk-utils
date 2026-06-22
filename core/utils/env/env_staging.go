//go:build staging

package env

// Staging environment settings hex encoded (points at the nexifi.ph staging cloud).
// RPC_PROXY_URL: https://cf-proxy.nexifi.ph
// SERVER_DOMAIN: nexifi.ph  (rpc/web URLs derived via RpcUpstreamURL/WebBaseURL)
// RPC_TOKEN: stg_uI9ZNQC0EfCtpezr0zqwASgk
//
// NOTE: RPC_TOKEN baked here must equal the staging host's .env RPC_TOKEN (machine ↔
// cloud auth). Staging uses a DISTINCT secret from production — prod (env.go) bakes a
// different token, so a leaked staging build can never authenticate against the prod
// cloud and vice-versa. To rotate: generate a new token, re-hex-encode it here, set the
// matching plaintext in the staging .env, update the builder bots' config, then rebuild
// and reflash the device (the value is compiled in, not read from the environment).

const (
	GO_ENV       int8   = ENV_STAGING
	HTTP_PORT    int    = 80
	HTTPS_PORT   int    = 443
	LocalBaseURL string = "http://127.0.0.1"

	// Hex encoded URL and token for obfuscation
	rpcProxyURLEncoded  string = "68747470733a2f2f63662d70726f78792e6e65786966692e7068"
	serverDomainEncoded string = "6e65786966692e7068"
	rpcTokenEncoded     string = "7374675f7549395a4e5143304566437470657a72307a71774153676b"
)

var (
	RPC_PROXY_URL string
	SERVER_DOMAIN string
	RPC_TOKEN     string
)

func init() {
	RPC_PROXY_URL = DecodeURL(rpcProxyURLEncoded)
	SERVER_DOMAIN = DecodeURL(serverDomainEncoded)
	RPC_TOKEN = DecodeURL(rpcTokenEncoded)
}
