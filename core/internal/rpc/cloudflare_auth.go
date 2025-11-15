package rpc_flarewifi_v1

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	machineuid "core/internal/utils/machine-uid"
)

// CloudflareAuth adds Cloudflare Worker validation headers to every request.
// It creates a JWT token signed with Machine-Id + MD5(body) and adds it as Payload-Hash header.
type CloudflareAuth struct {
	rt http.RoundTripper
}

// computeMD5 computes MD5 hash and returns it as lowercase base64 string.
// This matches the Cloudflare Worker's implementation which uses btoa() on the raw binary hash.
// Note: btoa() in JavaScript produces the same output as base64.StdEncoding in Go.
func computeMD5(message []byte) string {
	hash := md5.Sum(message)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// RoundTrip implements the http.RoundTripper interface.
// It adds Payload-Hash (JWT token) and Machine-Id headers before forwarding the request.
func (a *CloudflareAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get machine ID
	machineID := machineuid.GetMachineUID()

	// Read request body to compute hash
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body for the actual request
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Compute MD5 hash of request body
	bodyMD5 := computeMD5(body)

	// JWT secret is Machine-Id + MD5 of request body
	jwtSecret := machineID + bodyMD5

	// Create JWT token with 5-minute expiration
	now := time.Now()
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

	// Continue with the request using the wrapped RoundTripper
	return a.rt.RoundTrip(req)
}
