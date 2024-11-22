/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkfs

import (
	"os"
)

func EnsureDir(dirs ...string) error {
	for _, dir := range dirs {
		if !Exists(dir) {
			err := os.MkdirAll(dir, PermDir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
