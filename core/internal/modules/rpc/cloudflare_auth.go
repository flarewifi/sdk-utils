package rpcutil

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// NewCloudflareClient builds the http.Client used for every core->cloud RPC
// call: auth header injection layered on top of retry/timeout resiliency
// (see rpc_resilience.go) on top of the shared base transport.
func NewCloudflareClient(machineID string) *http.Client {
	tr := NewCloudflareRoundTripper(newResilientTransport(rpcTransport), machineID)
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
// It is solely responsible for auth header injection — retries, timeouts, and
// context-cancellation handling live in the wrapped RoundTripper (see
// rpc_resilience.go).
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

// RoundTrip implements the http.RoundTripper interface. It adds Payload-Hash
// (JWT token) and Machine-Id headers before forwarding the request to the
// wrapped RoundTripper.
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

	// Rearm the body (already fully buffered above) and expose GetBody so a
	// downstream retrying RoundTripper can re-read it on each attempt without
	// needing to know anything about how it was produced.
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}

	return a.rt.RoundTrip(req)
}
