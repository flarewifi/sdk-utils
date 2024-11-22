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

func MoveDir(sourceDir, destDir string) error {
	// Check if the source directory exists
	_, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}

	// Create the destination directory if it doesn't exist
	err = os.MkdirAll(destDir, 0755) // Directories with permission mode 0755
	if err != nil {
		return err
	}

	// Walk through the source directory and move files and subdirectories
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create the destination path for the current item
		destPath := filepath.Join(destDir, path[len(sourceDir):])

		if info.IsDir() {
			// Create the directory in the destination path with permission mode 0755
			err := os.MkdirAll(destPath, 0755) // Directories with permission mode 0755
			if err != nil {
				return err
			}
		} else {
			// Move the file to the destination path
			err := os.Rename(path, destPath)
			if err != nil {
				return err
			}

			// Set the permission mode for the moved file
			err = os.Chmod(destPath, 0644) // Files with permission mode 0644
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Remove the source directory after successfully moving its contents
	err = os.RemoveAll(sourceDir)
	if err != nil {
		return err
	}

	return nil
}

