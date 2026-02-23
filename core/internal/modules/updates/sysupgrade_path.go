//go:build !dev
// +build !dev

package updates

import (
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// getSysupgradeDir returns the directory path for sysupgrade files (production)
func getSysupgradeDir() string {
	return sdkutils.PathSysupgradeDir
}

// getSystemUpdateDir returns the directory path for system update marker files (production)
func getSystemUpdateDir() string {
	return sdkutils.PathSystemUpdateDir
}

// getDownloadDir returns the directory path for downloaded update files (production)
func getDownloadDir() string {
	return sdkutils.PathSystemUpdateDir
}

// ensureSysupgradeDirs ensures all required directories exist (production)
func ensureSysupgradeDirs() error {
	// Ensure the sysupgrade directory exists
	if err := sdkutils.FsEnsureDir(sdkutils.PathSysupgradeDir); err != nil {
		return err
	}

	// Ensure the updates directory exists for marker file
	if err := sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}

	return nil
}

// getDownloadedFilePath returns the full path for a downloaded update file (production)
func getDownloadedFilePath(filename string) string {
	return filepath.Join(sdkutils.PathSystemUpdateDir, filename)
}
