//go:build devkit

package rpc

import (
	"context"
	"errors"
	"net/http"

	"core/internal/rpc/rpc_flarewifi_v3"
)

const RPC_API_VERSION = "v3"

// errCloudDisabled is returned for any attempted core→cloud RPC in a devkit build.
var errCloudDisabled = errors.New("devkit: core cloud RPC is disabled")

// deadTransport fails every request without performing a network dial, so a devkit
// build can never contact — or even reference — the real server domain.
type deadTransport struct{}

func (deadTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errCloudDisabled
}

// GetTwirpServiceAndCtx is the devkit variant of the core's Twirp client factory.
// The devkit neutralizes all core call-home at this single choke point: the client
// is bound to an unroutable local address with a transport that always errors, so
// any accidental call from gated code fails fast and no server domain is embedded
// or contacted. Plugins are unaffected — they dial via their own clients.
func GetTwirpServiceAndCtx() (rpc_flarewifi_v3.FlarehotspotService, context.Context) {
	httpClient := &http.Client{Transport: deadTransport{}}
	srv := rpc_flarewifi_v3.NewFlarehotspotServiceProtobufClient("http://127.0.0.1:0", httpClient)
	return srv, context.Background()
}
