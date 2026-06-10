//go:build mono

// Mono system-update launcher: download the single system release tarball. The
// mono boot script (start-mono.sh) replaces the whole app from it on next boot.
package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
)

func startSystemDownload(g *api.CoreGlobals, update *updates.SoftwareReleaseUpdate) {
	go updates.DownloadSoftwareUpdate(updates.DownloadParams{
		FileURL:      update.ReleseFileURL,
		Checksum:     update.ReleaseFileChecksum,
		OutputPath:   updates.GetUpdateOutputPath(update.ReleseFileURL, update.IsSysupgrade),
		IsSysupgrade: update.IsSysupgrade,
	})
}
