package sdkutils

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var ErrChecksumVerificationFailed = errors.New("checksum verification failed")

// BadStatusError indicates the server returned a non-2xx status. It carries
// the status code so callers can tell a permanent failure (e.g. 404 for an
// invalid/expired URL) from a transient one worth retrying.
type BadStatusError struct {
	Code int
}

func (e *BadStatusError) Error() string {
	return "bad status: " + strconv.Itoa(e.Code)
}

// downloadHTTPClient is used for plugin archive/tarball transfers. It has no
// overall request timeout — a large archive on a slow link must be allowed to
// keep progressing — but bounds each connection-setup stage so a stuck dial,
// TLS handshake, or unresponsive server fails fast instead of hanging
// forever. Proxy is explicit (not implied by a zero-value Transport) so
// HTTP_PROXY/HTTPS_PROXY/NO_PROXY keep working, same as http.DefaultTransport.
// Keepalives are disabled to avoid stale connection issues, matching the
// pattern used by com.flarego.publisher's apiClient for the same kind of transfer.
var downloadHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DisableKeepAlives:     true,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

// DownloadWithProgressOpts contains optional parameters for downloading files
type DownloadWithProgressOpts struct {
	Md5Checksum string
}

func DownloadWithProgress(url, dest string, opts ...*DownloadWithProgressOpts) (<-chan int, <-chan error) {
	progress := make(chan int)
	errChan := make(chan error, 1)

	go func() {
		defer close(progress)
		defer close(errChan)

		// Extract options if provided
		var expectedChecksum string
		if len(opts) > 0 && opts[0] != nil {
			expectedChecksum = opts[0].Md5Checksum
		}

		resp, err := downloadHTTPClient.Get(url)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- &BadStatusError{Code: resp.StatusCode}
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
						errChan <- ErrChecksumVerificationFailed
						return
					}
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
