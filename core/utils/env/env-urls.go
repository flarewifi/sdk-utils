package env

// scheme returns the URL scheme for the current build: http in development,
// https everywhere else.
func scheme() string {
	if GO_ENV == ENV_DEV {
		return "http"
	}
	return "https"
}

// RpcUpstreamURL is the real RPC server behind the Cloudflare proxy (the proxy
// forwards requests here via the Forward-To header). Derived from SERVER_DOMAIN:
// https://rpc.flarewifi.com (prod), http://rpc.flare-local.com (dev).
func RpcUpstreamURL() string {
	return scheme() + "://rpc." + SERVER_DOMAIN
}

// WebBaseURL is the cloud dashboard's web origin (where pages like the plugin
// checkout live). Derived from SERVER_DOMAIN: https://www.flarewifi.com (prod),
// http://www.flare-local.com (dev).
func WebBaseURL() string {
	return scheme() + "://www." + SERVER_DOMAIN
}

// SiteURL is the marketing/site origin for this build's cloud, derived from
// SERVER_DOMAIN: https://flarewifi.com (prod), https://nexifi.ph (staging),
// http://flare-local.com (dev). Used for "Powered by" / brand links so a
// staging machine never links out to the production site.
func SiteURL() string {
	return scheme() + "://" + SERVER_DOMAIN
}

// PortalDomain is the shared captive-portal hostname for this build's cloud,
// derived from SERVER_DOMAIN: captive.flare-local.com (dev), captive.nexifi.ph
// (staging), captive.flarewifi.com (prod). The cloud issues the portal TLS
// certificate for exactly this hostname, so dev and staging machines (which
// carry no per-machine custom_domain) funnel portal/captive traffic here — via
// the HTTPS redirect and split-horizon DNS — to present a valid cert.
func PortalDomain() string {
	return "captive." + SERVER_DOMAIN
}
