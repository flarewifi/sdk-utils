package rpc_flarewifi_v1

import (
	"context"
	machineuid "core/internal/utils/machine-uid"
	rpcutil "core/internal/utils/rpc"
	"log"
	"net/http"
	"tools/env"

	"github.com/twitchtv/twirp"
)

func GetTwirpServiceAndCtx() (FlarehotspotService, context.Context) {
	url := env.RPC_PROXY_URL + "/flarewifi/" + env.RPC_API_VERSION

	// Create HTTP client with custom RoundTripper for Cloudflare Worker validation
	machineID := machineuid.GetMachineUID()
	httpClient := rpcutil.NewCloudflareClient(machineID)
	srv := NewFlarehotspotServiceProtobufClient(url, httpClient)
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
