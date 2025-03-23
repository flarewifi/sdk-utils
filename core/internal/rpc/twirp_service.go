package rpc

import (
	"context"
	"core/env"
	"log"
	"net/http"

	"github.com/twitchtv/twirp"
)

func GetCoreTwirpServiceAndCtx() (FlarehotspotService, context.Context) {
	isDev := true

	proto := "http"
	prefix := "v0.0.1"
	domain := "flarehotspot.com"
	subdomain := "rpc-machines"

	if isDev {
		domain = "flarehotspot-dev.com"
	}

	baseUrl := subdomain + "." + domain
	url := proto + "://" + baseUrl + "/" + prefix

	srv := NewFlarehotspotServiceProtobufClient(url, &http.Client{})
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+env.RPC_TOKEN)

	ctx := context.Background()
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, header)
	if err != nil {
		log.Fatalf("twirp error setting headers: %s", err)
	}

	return srv, ctx
}
