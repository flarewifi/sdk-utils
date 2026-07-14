package rpcutil

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// dialTimeout bounds a TCP connect to the cloud RPC endpoint. Go's
	// http.DefaultTransport default is 30s; widened here since the machine's
	// uplink (mobile/satellite backhaul on some sites) can be slow to establish
	// a connection even when it will ultimately succeed.
	dialTimeout = 60 * time.Second

	// rpcRetryAttempts applies to every internal core->cloud RPC call, since all
	// of them funnel through this one RoundTripper — a single choke point means
	// no individual call site needs its own retry logic. A retry only ever fires
	// for a failure confirmed (via httptrace) to have happened BEFORE the request
	// was fully written to the wire — see RoundTrip — so a request the server may
	// already have received and processed (even one that comes back as a network
	// error while we're awaiting/reading the response) is never resent. That
	// matters because several RPCs behind this client (e.g. VerifyOtp, GenerateOtp)
	// are not idempotent: blindly resending them after an ambiguous failure could
	// re-consume a one-time code or double-send an OTP email.
	rpcRetryAttempts = 3

	// rpcCallBudget bounds the TOTAL wall-clock time a single RoundTrip (across all
	// of its internal retries) may take, independent of dialTimeout. Without this,
	// rpcRetryAttempts*dialTimeout can blow up (e.g. 3*60s = 180s) and compound with
	// a caller's OWN outer retry loop (activation.go retries RPC calls up to 5x),
	// turning a single logical operation into 15+ minutes of blocking during a cloud
	// outage. Capping the budget here keeps that compounding bounded and predictable
	// regardless of how dialTimeout or rpcRetryAttempts are tuned later.
	rpcCallBudget = 90 * time.Second
)

// rpcTransport is the shared base transport for all core->cloud RPC clients,
// identical to http.DefaultTransport except for dialTimeout.
var rpcTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

func NewCloudflareClient(machineID string) *http.Client {
	tr := NewCloudflareRoundTripper(rpcTransport, machineID)
	httpClient := &http.Client{
		Transport: tr,
	}
	return httpClient
}

func NewCloudflareRoundTripper(rt http.RoundTripper, machineID string) *CloudflareAuth {
	return &CloudflareAuth{rt: rt, machineID: machineID}
}

// CloudflareAuth adds Cloudflare Worker validation headers to every request.
// It creates a JWT token signed with Machine-Id + MD5(body) and adds it as Payload-Hash header.
type CloudflareAuth struct {
	rt        http.RoundTripper
	machineID string
}

// computeMD5 computes MD5 hash and returns it as lowercase base64 string.
// This matches the Cloudflare Worker's implementation which uses btoa() on the raw binary hash.
// Note: btoa() in JavaScript produces the same output as base64.StdEncoding in Go.
func computeMD5(message []byte) string {
	hash := md5.Sum(message)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// RoundTrip implements the http.RoundTripper interface.
// It adds Payload-Hash (JWT token) and Machine-Id headers before forwarding the
// request, then retries purely-pre-send failures (dial/connect errors that never
// reached the server) up to rpcRetryAttempts times, bounded overall by
// rpcCallBudget. A failure confirmed to have happened AFTER the request was fully
// written — including a network error while awaiting/reading the response — is
// returned immediately without a retry, since the RPC may already have been
// processed server-side and may not be safe to resend.
func (a *CloudflareAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get machine ID
	machineID := a.machineID

	// Read request body to compute hash
	var body []byte
	var err error

	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	// Compute MD5 hash of request body
	bodyMD5 := computeMD5(body)

	// JWT secret is Machine-Id + MD5 of request body
	jwtSecret := machineID + bodyMD5

	// Create JWT token with 5-minute expiration (use UTC for consistency)
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	// Add required headers for Cloudflare Worker validation
	req.Header.Set("Payload-Hash", tokenString)
	req.Header.Set("Machine-Id", machineID)

	ctx, cancel := context.WithTimeout(req.Context(), rpcCallBudget)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= rpcRetryAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Restore the body fresh on every attempt — a prior attempt's RoundTrip
		// (or a dial failure mid-write) may have already consumed the reader.
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		sent := false
		trace := &httptrace.ClientTrace{
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				sent = info.Err == nil
			},
		}
		attemptReq := req.WithContext(httptrace.WithClientTrace(ctx, trace))

		resp, err := a.rt.RoundTrip(attemptReq)
		if err == nil {
			return resp, nil
		}

		if sent {
			return nil, err
		}

		lastErr = err
		if attempt < rpcRetryAttempts {
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("rpc: after %d attempts, last error: %w", rpcRetryAttempts, lastErr)
}
