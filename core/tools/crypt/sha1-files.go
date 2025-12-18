package crypt

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

func SHA1Files(files ...string) (string, error) {
	hash := sha1.New()

	for _, f := range files {
		sha1File(f, hash)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha1File(f string, hash hash.Hash) error {
	file, err := os.Open(f)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(hash, file)

	if err != nil {
		return err
	}

	return nil
}
