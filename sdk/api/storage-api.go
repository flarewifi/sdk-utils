/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "io"

// IStorageApi provides file storage operations for plugins.
// Files are stored in data/storage/plugins/{plugin-package}/{filename}
type IStorageApi interface {
	// Write writes bytes to a file in the plugin's storage directory.
	// Returns the absolute filesystem path to the stored file.
	// Returns error if data exceeds configured PluginMaxFileSize.
	Write(filename string, data []byte) (string, error)

	// Read reads bytes from a file in the plugin's storage directory.
	Read(filename string) ([]byte, error)

	// WriteReader writes from an io.Reader to a file (useful for HTTP uploads).
	// Returns the absolute filesystem path to the stored file.
	// Returns error if data exceeds configured PluginMaxFileSize.
	WriteReader(filename string, reader io.Reader) (string, error)

	// ReadReader returns an io.ReadCloser for streaming large files.
	// Caller is responsible for closing the reader.
	ReadReader(filename string) (io.ReadCloser, error)

	// Delete removes a file from the plugin's storage directory.
	// Automatically cleans up empty parent directories.
	Delete(filename string) error

	// Exists checks if a file exists in the plugin's storage directory.
	Exists(filename string) bool

	// Move renames or moves a file within the plugin's storage directory.
	// Both oldFilename and newFilename must be within the storage directory.
	// If destination exists, it will be overwritten.
	// Automatically cleans up empty parent directories from old location.
	Move(oldFilename string, newFilename string) error

	// List returns filenames in the plugin's storage directory (recursive).
	// Pattern uses filepath.Glob syntax supporting wildcards:
	//   - "*" matches any sequence of characters
	//   - "?" matches any single character
	//   - "[range]" matches character ranges
	//   - "**" in pattern enables recursive directory matching
	// Examples: "*.png", "images/**/*.jpg", "config.*"
	// If pattern is empty string, all files are returned.
	List(pattern string) ([]string, error)

	// Path returns the absolute filesystem path for a filename.
	Path(filename string) string

	// UrlFor returns the HTTP URL to access the stored file.
	// Example: /storage/plugin/com.flarego.wifi-hotspot/logo.png
	UrlFor(filename string) string
}
