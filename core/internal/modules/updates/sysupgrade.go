package updates

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cmd "core/tools/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	// SysupgradePath is the OpenWRT standard path for sysupgrade files
	SysupgradePath = "/tmp/sysupgrade.bin"

	// MaxSysupgradeFileSize is the maximum allowed file size (100 MB)
	MaxSysupgradeFileSize = 100 << 20
)

var (
	// Allowed sysupgrade file extensions
	allowedExtensions = []string{".bin", ".img"}

	// Sysupgrade errors
	ErrInvalidFileExtension     = errors.New("invalid file extension")
	ErrFileTooLarge             = errors.New("file size exceeds maximum allowed")
	ErrSaveFile                 = errors.New("failed to save sysupgrade file")
	ErrIncompatibleFirmware     = errors.New("firmware is not compatible with this device")
	ErrFirmwareValidationFailed = errors.New("firmware validation failed")
)

// ValidateSysupgradeFile validates the uploaded sysupgrade file
// Returns nil if valid, error otherwise
func ValidateSysupgradeFile(filename string, fileSize int64) error {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))
	validExt := slices.Contains(allowedExtensions, ext)
	if !validExt {
		return ErrInvalidFileExtension
	}

	// Check file size
	if fileSize > MaxSysupgradeFileSize {
		return ErrFileTooLarge
	}

	return nil
}

// SaveSysupgradeFile saves the uploaded sysupgrade file to /tmp/sysupgrade.bin
// (OpenWRT standard path) and creates the completion marker file
func SaveSysupgradeFile(src io.Reader, filename string) error {
	// Create the destination file at OpenWRT standard sysupgrade path
	destFile, err := os.Create(SysupgradePath)
	if err != nil {
		return ErrSaveFile
	}
	defer destFile.Close()

	// Copy the file contents
	if _, err := io.Copy(destFile, src); err != nil {
		os.Remove(SysupgradePath)
		return ErrSaveFile
	}

	// Ensure the marker directory exists
	if err := sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir); err != nil {
		return ErrSaveFile
	}

	// Create completion marker file for UI state tracking
	markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
	if err := os.WriteFile(markerPath, []byte("sysupgrade"), 0644); err != nil {
		os.Remove(SysupgradePath)
		return ErrSaveFile
	}

	return nil
}

// GetAllowedExtensions returns the list of allowed file extensions
func GetAllowedExtensions() []string {
	return allowedExtensions
}

// GetMaxFileSizeMB returns the maximum file size in megabytes
func GetMaxFileSizeMB() int {
	return MaxSysupgradeFileSize >> 20
}

// GetSysupgradePath returns the path where sysupgrade file is saved
func GetSysupgradePath() string {
	return SysupgradePath
}

// IsSysupgradeReady checks if a sysupgrade file is ready at /tmp/sysupgrade.bin
func IsSysupgradeReady() bool {
	return sdkutils.FsExists(SysupgradePath)
}

// ValidateSysupgradeCompatibility runs sysupgrade -T to check if the firmware
// is compatible with the current device. Returns nil if compatible.
func ValidateSysupgradeCompatibility() error {
	if !IsSysupgradeReady() {
		return ErrFirmwareValidationFailed
	}

	// Run sysupgrade -T to test firmware compatibility
	// Exit code 0 = compatible, non-zero = incompatible
	err := cmd.Exec("sysupgrade -T "+SysupgradePath, nil)
	if err != nil {
		return ErrIncompatibleFirmware
	}

	return nil
}

// RemoveSysupgradeFile removes the sysupgrade file and marker
func RemoveSysupgradeFile() {
	os.Remove(SysupgradePath)
	markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
	os.Remove(markerPath)
}
