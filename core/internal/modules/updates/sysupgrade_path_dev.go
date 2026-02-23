//go:build dev
// +build dev

package updates

import (
	"os"
	"path/filepath"
)

// getSysupgradeDir returns the directory path for sysupgrade files (dev - uses /tmp/sysupgrade)
func getSysupgradeDir() string {
	return "/tmp/sysupgrade"
}

// getSystemUpdateDir returns the directory path for system update marker files (dev - uses /tmp/sysupgrade/updates)
func getSystemUpdateDir() string {
	return "/tmp/sysupgrade/updates"
}

// getDownloadDir returns the directory path for downloaded update files (dev - uses /tmp/sysupgrade/updates)
func getDownloadDir() string {
	return "/tmp/sysupgrade/updates"
}

// ensureSysupgradeDirs ensures all required directories exist (dev)
func ensureSysupgradeDirs() error {
	// Ensure the sysupgrade directory exists
	sysupgradeDir := getSysupgradeDir()
	if err := os.MkdirAll(sysupgradeDir, 0755); err != nil {
		return err
	}

	// Ensure the updates subdirectory exists
	updateDir := getSystemUpdateDir()
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		return err
	}

	return nil
}

// getDownloadedFilePath returns the full path for a downloaded update file (dev)
func getDownloadedFilePath(filename string) string {
	return filepath.Join(getDownloadDir(), filename)
}
