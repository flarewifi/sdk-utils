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

// LsFiles returns list if files within dir. File paths are prepended with dir. It follows symlinks.
func LsFiles(dir string, files *[]string, recursive bool) error {
	stat, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(dir)
		if err != nil {
			return err
		}

		dir = target
	}

	fileEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range fileEntries {
		entryPath := filepath.Join(dir, entry.Name())
		if IsDir(entryPath) {
			if recursive {
				err = LsFiles(entryPath, files, recursive)
				if err != nil {
					return err
				}
			}
		} else {
			*files = append(*files, entryPath)
		}
	}

	return nil
}
