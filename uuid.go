package sdkutils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// NewUUID generates a new random UUID string
func NewUUID() string {
	return uuid.New().String()
}

// HashUUID generates a deterministic UUID from input strings using SHA256.
// Returns a properly formatted UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// The UUID is derived from the first 32 hex characters of the SHA256 hash.
// Useful for creating stable, unique identifiers for payment providers, etc.
func HashUUID(inputs ...string) string {
	h := sha256.New()
	for _, input := range inputs {
		h.Write([]byte(input))
	}
	hash := hex.EncodeToString(h.Sum(nil))
	// Format as UUID: 8-4-4-4-12 (32 hex chars total)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hash[0:8],
		hash[8:12],
		hash[12:16],
		hash[16:20],
		hash[20:32],
	)
}
