package rpcutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
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

// cancelOnCloseBody defers releasing a RoundTrip's timeout context until the
// caller finishes reading the response — canceling any earlier (e.g. via a
// bare `defer cancel()` in RoundTrip) races the body read against
// RoundTrip's own return, since callers read/unmarshal the body AFTER
// RoundTrip hands it back, not before.
type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// resilientTransport wraps a RoundTripper with retry, timeout, and safe
// context-cancellation handling. It is the sole place core->cloud RPC calls
// get this behavior — callers (e.g. CloudflareAuth) just delegate to it.
type resilientTransport struct {
	rt http.RoundTripper
}

func newResilientTransport(rt http.RoundTripper) *resilientTransport {
	return &resilientTransport{rt: rt}
}

// RoundTrip implements the http.RoundTripper interface. It retries
// purely-pre-send failures (dial/connect errors that never reached the
// server) up to rpcRetryAttempts times, bounded overall by rpcCallBudget. A
// failure confirmed to have happened AFTER the request was fully written —
// including a network error while awaiting/reading the response — is
// returned immediately without a retry, since the RPC may already have been
// processed server-side and may not be safe to resend.
func (t *resilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), rpcCallBudget)
	// Only cancel here on a path that returns without a response body for the
	// caller to read. On success, cancellation is deferred to the wrapped
	// body's Close (below) — canceling immediately on RoundTrip's return would
	// tear down the response body mid-read, since the caller reads/unmarshals
	// it AFTER RoundTrip returns, not before.
	cancelOnReturn := true
	defer func() {
		if cancelOnReturn {
			cancel()
		}
	}()

	var lastErr error
	for attempt := 1; attempt <= rpcRetryAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		sent := false
		trace := &httptrace.ClientTrace{
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				sent = info.Err == nil
			},
		}
		// A fresh request per attempt — a prior attempt's RoundTrip (or a dial
		// failure mid-write) may have already consumed its body reader, so
		// cloning off the original (never-consumed) req and rearming its body
		// avoids mutating shared state across attempts.
		attemptReq := req.Clone(httptrace.WithClientTrace(ctx, trace))
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq.Body = body
		}

		resp, err := t.rt.RoundTrip(attemptReq)
		if err == nil {
			cancelOnReturn = false
			resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
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
