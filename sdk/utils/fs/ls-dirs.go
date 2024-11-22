/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkfs

import (
	"os"
	"path/filepath"
)

// LsDirs returns directories inside dir. Directory paths are prepended with parent directory path.
func LsDirs(path string, directories *[]string, recursive bool) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}

		path = target
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if IsDir(filepath.Join(path, entry.Name())) {
			*directories = append(*directories, filepath.Join(path, entry.Name()))

			if recursive {
				subdirPath := filepath.Join(path, entry.Name())
				err := LsDirs(subdirPath, directories, recursive)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// IsDir returns true if path is a directory. It follows symlinks.
func IsDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	if stat.IsDir() {
		return true // It's a directory
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return false // Error reading symbolic link target
		}

		targetInfo, err := os.Stat(target)
		if err != nil {
			return false // Error getting information about the target
		}

		return targetInfo.IsDir()
	}

	return false
}
