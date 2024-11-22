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

type FindFilterFn func(currDir string, entry string, stat os.FileInfo) bool
type FindReturnFn func(currDir string, entry string, stat os.FileInfo) string
type FindOpts struct {
	StopRecursion bool
}

func Find(dir string, filter FindFilterFn, ret FindReturnFn, opts FindOpts) []string {
	results := []string{}

	if Exists(dir) && IsDir(dir) {
		var files []string
		if err := LsFiles(dir, &files, false); err != nil {
			return results
		}

		for _, entry := range files {
			stat, err := os.Stat(entry)
			if err != nil {
				continue
			}
			entry = filepath.Base(entry)
			ok := filter(dir, entry, stat)
			if ok {
				results = append(results, ret(dir, entry, stat))
			}

			if ok && opts.StopRecursion {
				return results
			}
		}

		var dirs []string
		if err := LsDirs(dir, &dirs, false); err != nil {
			return results
		}

		for _, entry := range dirs {
			stat, err := os.Stat(entry)
			if err != nil {
				continue
			}
			entry = filepath.Base(entry)
			ok := filter(dir, entry, stat)
			if ok {
				results = append(results, ret(dir, entry, stat))
			}
			if ok && opts.StopRecursion {
				return results
			}
			subResults := Find(entry, filter, ret, opts)
			results = append(results, subResults...)
		}

		return results
	}

	return results

}
