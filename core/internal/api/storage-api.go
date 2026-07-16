package api

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

type StorageApi struct {
	api *PluginApi
}

func NewStorageApi(api *PluginApi) sdkapi.IStorageApi {
	return &StorageApi{api: api}
}

// Returns the storage directory for this plugin
func (s *StorageApi) storageDir() string {
	return filepath.Join(sdkutils.PathPluginStorageDir, s.api.info.Package)
}

// Sanitize filename to prevent path traversal attacks
func (s *StorageApi) sanitizePath(filename string) (string, error) {
	// Remove any leading slashes or backslashes
	filename = strings.TrimLeft(filename, "/\\")

	// Convert to forward slashes for consistency
	filename = filepath.ToSlash(filename)

	// Check for path traversal attempts
	if strings.Contains(filename, "..") {
		return "", errors.New("invalid filename: path traversal not allowed")
	}

	// Build full path and ensure it's within storage dir
	fullPath := filepath.Join(s.storageDir(), filename)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	absStorageDir, err := filepath.Abs(s.storageDir())
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absPath, absStorageDir) {
		return "", errors.New("invalid filename: must be within plugin storage directory")
	}

	return absPath, nil
}

// Clean up empty parent directories recursively
func (s *StorageApi) cleanupEmptyDirs(filePath string) error {
	dir := filepath.Dir(filePath)
	storageDir := s.storageDir()

	// Don't remove the storage root directory itself
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil // Silently fail cleanup
	}

	absStorageDir, err := filepath.Abs(storageDir)
	if err != nil {
		return nil
	}

	// If we're at or above storage dir, stop
	if absDir == absStorageDir || !strings.HasPrefix(absDir, absStorageDir) {
		return nil
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // Directory doesn't exist or can't read, that's fine
	}

	if len(entries) == 0 {
		// Remove empty directory
		if err := os.Remove(dir); err != nil {
			return nil // Silently fail
		}

		// Recursively check parent
		return s.cleanupEmptyDirs(dir)
	}

	return nil
}

func (s *StorageApi) getMaxFileSize() int64 {
	cfg, err := s.api.Config().Application().Get()
	if err != nil {
		return 10 * 1024 * 1024 // 10MB default
	}
	return cfg.PluginMaxFileSize
}

func (s *StorageApi) Write(filename string, data []byte) (string, error) {
	maxSize := s.getMaxFileSize()
	if int64(len(data)) > maxSize {
		return "", fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", len(data), maxSize)
	}

	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return "", err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := sdkutils.FsEnsureDir(dir); err != nil {
		return "", err
	}

	// Write file
	if err := os.WriteFile(fullPath, data, sdkutils.PermFile); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s *StorageApi) Read(filename string) ([]byte, error) {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *StorageApi) WriteReader(filename string, reader io.Reader) (string, error) {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return "", err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := sdkutils.FsEnsureDir(dir); err != nil {
		return "", err
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Copy with size limit
	maxSize := s.getMaxFileSize()
	written, err := io.CopyN(file, reader, maxSize+1)
	if err != nil && err != io.EOF {
		os.Remove(fullPath) // Clean up on error
		return "", err
	}

	if written > maxSize {
		os.Remove(fullPath)
		return "", fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxSize)
	}

	return fullPath, nil
}

func (s *StorageApi) ReadReader(filename string) (io.ReadCloser, error) {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s *StorageApi) Delete(filename string) error {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		return err
	}

	// Clean up empty parent directories
	s.cleanupEmptyDirs(fullPath)

	return nil
}

func (s *StorageApi) Exists(filename string) bool {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return false
	}

	return sdkutils.FsExists(fullPath)
}

func (s *StorageApi) Move(oldFilename string, newFilename string) error {
	oldPath, err := s.sanitizePath(oldFilename)
	if err != nil {
		return fmt.Errorf("invalid source filename: %w", err)
	}

	newPath, err := s.sanitizePath(newFilename)
	if err != nil {
		return fmt.Errorf("invalid destination filename: %w", err)
	}

	// Check if source exists
	if !sdkutils.FsExists(oldPath) {
		return fmt.Errorf("source file does not exist: %s", oldFilename)
	}

	// Ensure destination parent directory exists
	newDir := filepath.Dir(newPath)
	if err := sdkutils.FsEnsureDir(newDir); err != nil {
		return err
	}

	// If destination exists, remove it (overwrite behavior)
	if sdkutils.FsExists(newPath) {
		if err := os.Remove(newPath); err != nil {
			return fmt.Errorf("failed to overwrite destination: %w", err)
		}
	}

	// Move/rename file
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	// Clean up empty directories from old location
	s.cleanupEmptyDirs(oldPath)

	return nil
}

func (s *StorageApi) List(pattern string) ([]string, error) {
	storageDir := s.storageDir()

	if !sdkutils.FsExists(storageDir) {
		return []string{}, nil
	}

	var files []string

	// If no pattern, return all files
	if pattern == "" {
		err := filepath.Walk(storageDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(storageDir, path)
				if err != nil {
					return err
				}
				files = append(files, filepath.ToSlash(relPath))
			}
			return nil
		})
		return files, err
	}

	// Handle recursive pattern with **
	if strings.Contains(pattern, "**") {
		// Walk all directories and match pattern
		err := filepath.Walk(storageDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(storageDir, path)
				if err != nil {
					return err
				}
				relPath = filepath.ToSlash(relPath)

				// Try matching with the pattern
				matched, err := filepath.Match(strings.ReplaceAll(pattern, "**", "*"), relPath)
				if err != nil {
					return err
				}

				if matched {
					files = append(files, relPath)
				}
			}
			return nil
		})
		return files, err
	}

	// Use filepath.Glob for non-recursive patterns
	globPattern := filepath.Join(storageDir, pattern)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, err
	}

	// Convert absolute paths to relative paths
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(storageDir, match)
			if err != nil {
				continue
			}
			files = append(files, filepath.ToSlash(relPath))
		}
	}

	return files, nil
}

func (s *StorageApi) Path(filename string) string {
	fullPath, err := s.sanitizePath(filename)
	if err != nil {
		return ""
	}
	return fullPath
}

func (s *StorageApi) UrlFor(filename string) string {
	return path.Join("/storage/plugin", s.api.info.Package, filepath.ToSlash(filename))
}

// EnsureDir creates the plugin's storage directory if it does not already exist.
// Core calls this when the plugin API is initialized (PluginApi.Initialize) so the
// /storage/plugin/<pkg>/ HTTP route is always registered at boot: setupAssetsRoutes
// only mounts that route when the directory exists, so without pre-creation a file
// uploaded into a plugin that had never written to storage (e.g. a theme's first
// logo/banner upload) would 404 until the next restart.
func (s *StorageApi) EnsureDir() error {
	return os.MkdirAll(s.storageDir(), 0o755)
}
