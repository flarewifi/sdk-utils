package sdkutils

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var ErrChecksumVerificationFailed = errors.New("checksum verification failed")

// DownloadChOpts contains optional parameters for downloading files
type DownloadChOpts struct {
	Md5Checksum string
}

func DownloadCh(url, dest string, opts ...*DownloadChOpts) (<-chan int, <-chan error) {
	progress := make(chan int)
	errChan := make(chan error, 1)

	log.Println("Downloading", url, "to", dest)

	go func() {
		defer close(progress)
		defer close(errChan)

		// Extract options if provided
		var expectedChecksum string
		if len(opts) > 0 && opts[0] != nil {
			expectedChecksum = opts[0].Md5Checksum
		}

		resp, err := http.Get(url)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- errors.New("bad status: " + strconv.Itoa(resp.StatusCode))
			return
		}

		if err := os.MkdirAll(filepath.Dir(dest), PermDir); err != nil {
			errChan <- err
			return
		}

		file, err := os.Create(dest)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		totalSize := resp.ContentLength
		if totalSize <= 0 {
			errChan <- errors.New("unknown file size")
			return
		}

		// Create hash writer if checksum verification is needed
		var hasher hash.Hash
		var writer io.Writer = file
		if expectedChecksum != "" {
			hasher = md5.New()
			writer = io.MultiWriter(file, hasher)
		}

		written := int64(0)
		lastPercent := -1
		buf := make([]byte, 1024*8)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, writeErr := writer.Write(buf[:n])
				if writeErr != nil {
					errChan <- writeErr
					return
				}
				written += int64(n)
				currentPercent := int((written * 100) / totalSize)
				if currentPercent != lastPercent {
					progress <- currentPercent
					lastPercent = currentPercent
				}
			}
			if err == io.EOF {
				if written != totalSize {
					errChan <- errors.New("short read")
					return
				}

				// Verify checksum if provided
				if expectedChecksum != "" {
					actualChecksum := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
					if actualChecksum != expectedChecksum {
						log.Printf("Checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
						errChan <- ErrChecksumVerificationFailed
						return
					}
					log.Println("Checksum verified successfully")
				}

				errChan <- nil
				break
			}
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	return progress, errChan
}
