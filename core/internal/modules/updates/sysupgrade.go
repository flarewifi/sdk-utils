package updates

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cmd "core/utils/shell"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	// SysupgradeFilename is the filename for sysupgrade files
	SysupgradeFilename = "sysupgrade.bin"

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

// GetSysupgradePath returns the path where sysupgrade file is saved
// Production: data/storage/system/sysupgrade.bin
// Dev: /tmp/sysupgrade.bin
func GetSysupgradePath() string {
	return filepath.Join(getSysupgradeDir(), SysupgradeFilename)
}

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

// SaveSysupgradeFile saves the uploaded sysupgrade file to data/storage/system/sysupgrade.bin.
// After calling this, call FinalizeSysupgrade() to validate and create the completion marker.
func SaveSysupgradeFile(src io.Reader, filename string) error {
	// Ensure all required directories exist
	if err := ensureSysupgradeDirs(); err != nil {
		return ErrSaveFile
	}

	sysupgradePath := GetSysupgradePath()

	// Create the destination file
	destFile, err := os.Create(sysupgradePath)
	if err != nil {
		return ErrSaveFile
	}
	defer destFile.Close()

	// Copy the file contents
	if _, err := io.Copy(destFile, src); err != nil {
		os.Remove(sysupgradePath)
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

// IsSysupgradeReady checks if a sysupgrade file is ready
func IsSysupgradeReady() bool {
	return sdkutils.FsExists(GetSysupgradePath())
}

// ValidateSysupgradeCompatibility runs sysupgrade -T to check if the firmware
// is compatible with the current device. Returns nil if compatible.
func ValidateSysupgradeCompatibility() error {
	if !IsSysupgradeReady() {
		return ErrFirmwareValidationFailed
	}

	// Run sysupgrade -T to test firmware compatibility
	// Exit code 0 = compatible, non-zero = incompatible
	err := cmd.Exec("sysupgrade -T "+GetSysupgradePath(), nil)
	if err != nil {
		return ErrIncompatibleFirmware
	}

	return nil
}

// FinalizeSysupgrade validates the sysupgrade file compatibility and creates
// the completion marker. This is the shared path for both local uploads and
// remote downloads. Returns nil on success, removes the file on failure.
func FinalizeSysupgrade() error {
	// Validate firmware compatibility with the device
	if err := ValidateSysupgradeCompatibility(); err != nil {
		RemoveSysupgradeFile()
		return err
	}

	// Ensure the marker directory exists
	if err := sdkutils.FsEnsureDir(getSystemUpdateDir()); err != nil {
		RemoveSysupgradeFile()
		return ErrSaveFile
	}

	// Create completion marker file for UI state tracking
	markerPath := filepath.Join(getSystemUpdateDir(), downloadCompleteFile)
	if err := os.WriteFile(markerPath, []byte("sysupgrade"), 0644); err != nil {
		RemoveSysupgradeFile()
		return ErrSaveFile
	}

	return nil
}

// RemoveSysupgradeFile removes the sysupgrade file and marker
func RemoveSysupgradeFile() {
	os.Remove(GetSysupgradePath())
	markerPath := filepath.Join(getSystemUpdateDir(), downloadCompleteFile)
	os.Remove(markerPath)
}

// GetSysupgradeCommand returns the full sysupgrade command with appropriate flags
// noPreserve=true means use -n flag (do not preserve data)
func GetSysupgradeCommand(noPreserve bool) string {
	cmd := "sysupgrade"
	if noPreserve {
		cmd += " -n"
	}
	cmd += " " + GetSysupgradePath()
	return cmd
}
