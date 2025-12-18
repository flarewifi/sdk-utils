package crypt

import (
	"crypto/md5"
	"encoding/hex"
	"os"
)

// Hash files based on last modified time
func FastHashFiles(files ...string) (string, error) {
	hash := md5.New()
	for _, f := range files {
		if stat, err := os.Stat(f); err == nil {
			// How to get ctime:
			// https://stackoverflow.com/a/25164194/2441641
			ctime := stat.ModTime()
			hash.Write([]byte(f))
			hash.Write([]byte(ctime.String()))
		} else {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
