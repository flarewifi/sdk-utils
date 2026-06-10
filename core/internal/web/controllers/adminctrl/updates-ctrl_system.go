//go:build !mono

// Non-mono system-update launcher.
//
// A regular non-mono app update is staged as a core + ABI-matched plugin set
// (model A) via the orchestrator. A firmware sysupgrade is an OS-level image flash
// (orthogonal to mono/non-mono), so it keeps the plain file-download path that the
// sysupgrade success/flash flow already expects.
package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
)

func startSystemDownload(g *api.CoreGlobals, update *updates.SoftwareReleaseUpdate) {
	if update.IsSysupgrade {
		go updates.DownloadSoftwareUpdate(updates.DownloadParams{
			FileURL:      update.ReleseFileURL,
			Checksum:     update.ReleaseFileChecksum,
			OutputPath:   updates.GetUpdateOutputPath(update.ReleseFileURL, true),
			IsSysupgrade: true,
		})
		return
	}

	go updates.StageSystemUpdate(g, update)
}
