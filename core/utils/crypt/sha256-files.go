package crypt

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256File returns the lowercase hex SHA-256 of a single file, streaming it
// through the hash so a large plugin.so never loads fully into memory. Unlike
// SHA1Files it surfaces read/open errors to the caller (a missing file must not
// silently hash to the empty digest).
func SHA256File(f string) (string, error) {
	file, err := os.Open(f)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
