//go:build mono

// Mono local software-release apply.
//
// A mono build ships the core and every plugin as ONE tarball; start-mono.sh extracts
// it over the app dir and swaps the whole application on the next boot. The local
// apply therefore mirrors the cloud DownloadSoftwareUpdate path exactly, minus the
// download: drop the uploaded tarball into the system-updates dir and write the
// download-complete marker. IsDownloaded() (mono) reports that marker, so the caller
// routes into the existing download-done → reboot flow.
package updates

import (
	"core/internal/api"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// StageLocalSoftwareRelease places an already-saved software-release tarball
// (srcPath) where start-mono.sh applies it on the next reboot. _g is unused on mono
// (no per-plugin staging) but keeps the signature identical across build tags.
func StageLocalSoftwareRelease(_g *api.CoreGlobals, srcPath string) error {
	// Start from a clean staging root and clear any pending firmware image — a
	// sysupgrade and a software update are mutually exclusive.
	if err := sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}
	RemoveSysupgradeFile()

	// start-mono.sh globs $SOFTWARE_UPDATE_DIR/*.tar.gz, so the exact name is
	// immaterial; a fixed name keeps the dir tidy.
	dest := filepath.Join(sdkutils.PathSystemUpdateDir, "software-release.tar.gz")
	if err := sdkutils.FsCopyFile(srcPath, dest); err != nil {
		return ErrSaveFile
	}

	// Commit last: start-mono.sh only applies the tarball when this marker is present.
	markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
	if err := os.WriteFile(markerPath, []byte("complete"), 0644); err != nil {
		os.Remove(dest)
		return ErrSaveFile
	}

	return nil
}
