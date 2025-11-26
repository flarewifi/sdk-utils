package env

import "encoding/hex"

// DecodeURL decodes a hex-encoded string back to its original URL or string value.
// This is used for obfuscating sensitive URLs and tokens in production builds.
func DecodeURL(encoded string) string {
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		panic("Failed to decode URL: " + err.Error())
	}
	return string(decoded)
}
