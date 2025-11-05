package sdkutils

// Download is a simple synchronous wrapper around DownloadCh.
// It downloads a file from srcUrl to destPath without progress tracking.
func Download(srcUrl, destPath string) error {
	_, errChan := DownloadCh(srcUrl, destPath)
	return <-errChan
}
