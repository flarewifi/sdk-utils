package sdkutils

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// downloadRetries is the number of attempts Download makes before giving up,
// covering transient failures like connection resets or a stalled transfer.
const downloadRetries = 3

// Download is a simple synchronous wrapper around DownloadWithProgress.
// It downloads a file from srcUrl to destPath without progress tracking,
// retrying with backoff on transient network failures. A permanent failure
// (e.g. a 404 for an invalid/expired URL) is returned immediately since
// retrying it can never succeed.
func Download(srcUrl, destPath string) error {
	var lastErr error
	for attempt := 1; attempt <= downloadRetries; attempt++ {
		ch, errChan := DownloadWithProgress(srcUrl, destPath)
		for range ch {
		} // Dummy listener
		lastErr = <-errChan
		if lastErr == nil {
			return nil
		}
		if !isRetryableDownloadErr(lastErr) {
			return lastErr
		}
		if attempt < downloadRetries {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}
	return fmt.Errorf("after %d retries, last error: %w", downloadRetries, lastErr)
}

// isRetryableDownloadErr reports whether a failed download is worth retrying.
// A 4xx status (other than 429 Too Many Requests) means the request itself is
// invalid, so retrying against the same URL can never succeed.
func isRetryableDownloadErr(err error) bool {
	var badStatus *BadStatusError
	if errors.As(err, &badStatus) {
		return badStatus.Code == http.StatusTooManyRequests || badStatus.Code >= 500
	}
	return true
}
