//go:build mono

package updates

import "core/internal/api"

// startForcedUpdate kicks off applying a scheduled/forced update for a mono
// build: always a single plain file download (the whole app tarball, or a
// sysupgrade image) — mono has no separate staging pipeline. Mirrors adminctrl's
// mono startSystemDownload.
func startForcedUpdate(g *api.CoreGlobals, update *SoftwareReleaseUpdate) {
	DownloadSoftwareUpdate(DownloadParams{
		FileURL:      update.ReleseFileURL,
		Checksum:     update.ReleaseFileChecksum,
		OutputPath:   GetUpdateOutputPath(update.ReleseFileURL, update.IsSysupgrade),
		IsSysupgrade: update.IsSysupgrade,
	})
}
