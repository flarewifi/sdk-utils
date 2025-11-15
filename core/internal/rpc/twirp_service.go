package rpc_flarewifi_v1

import (
	"context"
	"log"
	"net/http"
	"tools/env"

	"github.com/twitchtv/twirp"
)

func GetTwirpServiceAndCtx() (FlarehotspotService, context.Context) {
	url := env.RPC_BASE_URL + "/flarewifi/" + env.RPC_API_VERSION

	// Create HTTP client with custom RoundTripper for Cloudflare Worker validation
	httpClient := &http.Client{
		Transport: &CloudflareAuth{
			rt: http.DefaultTransport,
		},
	}

	srv := NewFlarehotspotServiceProtobufClient(url, httpClient)
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+env.RPC_TOKEN)

	ctx := context.Background()
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, header)
	if err != nil {
		log.Fatalf("twirp error setting headers: %s", err)
	}

	return srv, ctx
}
