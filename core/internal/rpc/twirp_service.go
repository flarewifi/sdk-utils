//go:build !devkit

package rpc

import (
	"context"
	"fmt"
	machineuid "core/internal/modules/machine-uid"
	rpcutil "core/internal/modules/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/env"
	"net/http"

	"github.com/twitchtv/twirp"
)

const RPC_API_VERSION = "v3"

func GetTwirpServiceAndCtx() (rpc_flarewifi_v3.FlarehotspotService, context.Context) {
	url := env.RPC_PROXY_URL + "/flarewifi/" + RPC_API_VERSION

	// Create HTTP client with custom RoundTripper for Cloudflare Worker validation
	_, machineID := machineuid.GetMachineUID()
	httpClient := rpcutil.NewCloudflareClient(machineID)
	srv := rpc_flarewifi_v3.NewFlarehotspotServiceProtobufClient(url, httpClient)
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+env.RPC_TOKEN)
	header.Set("Forward-To", env.RpcUpstreamURL())

	ctx := context.Background()
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, header)
	if err != nil {
		panic(fmt.Errorf("twirp error setting headers: %s", err))
	}

	return srv, ctx
}
