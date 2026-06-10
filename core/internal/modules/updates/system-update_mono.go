//go:build mono

// Mono system-update completion gate.
//
// A mono build ships the core and all plugins as ONE tarball. downloadFile (in
// updates.go) writes the .dl_software_update_complete marker once that tarball is
// downloaded and verified, and the mono boot script (start-mono.sh) replaces the
// whole app from it on the next boot. IsDownloaded simply reports that marker.
package updates

import (
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func IsDownloaded() bool {
	markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
	return sdkutils.FsExists(markerPath)
}
