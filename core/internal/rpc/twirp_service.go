package rpc

import (
	"context"
	machineuid "core/internal/modules/machine-uid"
	rpcutil "core/internal/modules/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/env"
	"log"
	"net/http"

	"github.com/twitchtv/twirp"
)

const RPC_API_VERSION = "v2"

func GetTwirpServiceAndCtx() (rpc_flarewifi_v2.FlarehotspotService, context.Context) {
	url := env.RPC_PROXY_URL + "/flarewifi/" + RPC_API_VERSION

	// Create HTTP client with custom RoundTripper for Cloudflare Worker validation
	_, machineID := machineuid.GetMachineUID()
	httpClient := rpcutil.NewCloudflareClient(machineID)
	srv := rpc_flarewifi_v2.NewFlarehotspotServiceProtobufClient(url, httpClient)
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+env.RPC_TOKEN)
	header.Set("Forward-To", env.RPC_UPSTREAM_URL)

	ctx := context.Background()
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, header)
	if err != nil {
		log.Fatalf("twirp error setting headers: %s", err)
	}

	return srv, ctx
}
