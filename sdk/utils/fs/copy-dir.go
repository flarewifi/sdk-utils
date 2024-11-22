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

type CopyOpts struct {
	NoOverride   bool
	NonRecursive bool
}

func CopyDir(srcDir, destDir string, opts *CopyOpts) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &CopyOpts{}
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if opts.NoOverride {
			if _, err := os.Stat(destPath); err == nil {
				continue
			}
		}

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			continue
		}

		dir := filepath.Dir(destPath)
		if err := EnsureDir(dir); err != nil {
			continue
		}

		if entry.IsDir() && !opts.NonRecursive {
			if err := CopyDir(sourcePath, destPath, opts); err != nil {
				continue
			}
		} else if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			if err := CopySymLink(sourcePath, destPath); err != nil {
				continue
			}
		} else {
			if err := CopyFile(sourcePath, destPath); err != nil {
				continue
			}
		}
	}

	return nil
}
