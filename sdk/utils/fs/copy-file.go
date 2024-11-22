/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkfs

import (
	"io"
	"os"
	"path/filepath"
)

func CopyFile(srcFile, dstFile string) error {
	EnsureDir(filepath.Dir(dstFile))

	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	srcStat, err := os.Stat(srcFile)
	if err != nil {
		return err
	}

	err = os.Chmod(dstFile, srcStat.Mode())
	if err != nil {
		return err
	}

	return nil
}
