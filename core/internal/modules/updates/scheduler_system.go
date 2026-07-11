//go:build !mono

package updates

import "core/internal/api"

// startForcedUpdate kicks off applying a scheduled/forced update for a non-mono
// build. Mirrors adminctrl's startSystemDownload (the admin "Update Now"
// launcher): a sysupgrade is a plain OS-level image download (orthogonal to
// mono/non-mono, same as the mono variant below), but an ordinary core update
// must go through the two-phase staging pipeline (core extract + ABI-matched
// plugin rebuild via StageSystemUpdate) — never the generic single-file download
// mono uses for everything, whose completion marker non-mono's own IsDownloaded()
// never recognizes.
func startForcedUpdate(g *api.CoreGlobals, update *SoftwareReleaseUpdate) {
	if update.IsSysupgrade {
		DownloadSoftwareUpdate(DownloadParams{
			FileURL:      update.ReleseFileURL,
			Checksum:     update.ReleaseFileChecksum,
			OutputPath:   GetUpdateOutputPath(update.ReleseFileURL, true),
			IsSysupgrade: true,
		})
		return
	}
	StageSystemUpdate(g, update)
}
