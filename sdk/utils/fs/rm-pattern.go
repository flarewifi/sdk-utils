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

func RmPattern(dirPath string, globPattern string) error {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			matched, matchErr := filepath.Match(globPattern, info.Name())
			if matchErr != nil {
				return matchErr
			}

			if matched {
				removeErr := os.Remove(path)
				if removeErr != nil {
					return removeErr
				}
			}
		}
		return nil
	})

	return err
}

