//go:build devkit

package env

// Devkit builds run in development mode (GO_ENV == ENV_DEV) but with the core's
// cloud endpoints neutralized: no RPC proxy URL or token is embedded, and
// SERVER_DOMAIN is a local placeholder, so no real server domain is baked into
// the binary or derivable from it (RpcUpstreamURL/WebBaseURL/PortalDomain). The
// core never dials home anyway — see the rpc devkit variant
// (twirp_service_devkit.go). These values exist only to satisfy the URL helpers.
// Plugins are unaffected; they carry their own configuration.
const (
	GO_ENV    int8 = ENV_DEV
	HTTP_PORT int  = 3000
	// Matches the in-repo dev stack's HTTPS_PORT (env_dev.go) so a devkit plugin
	// author sees the same https://localhost:3443 admin URL as core contributors.
	HTTPS_PORT int = 3443
	// HTTPS-consistent base: ForceHTTPS serves the admin/portal over TLS (the
	// self-signed https://localhost:3443), so absolute URLs built from this base —
	// notably the login form action via UrlForRoute("auth:login") — must also be
	// https. With an http base the form posts cross-scheme from the https page and
	// Chromium blocks it as an insecure form submission.
	LocalBaseURL  string = "https://localhost:3443"
	RPC_TOKEN     string = ""
	RPC_PROXY_URL string = ""
	SERVER_DOMAIN string = "localhost"
)
