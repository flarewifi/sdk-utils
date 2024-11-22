/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkpaths

import (
	"path/filepath"
	"strings"
)

/*
returns the relative path from file "from" to file "to".

		Example:

		from := "/path/to/dir1/file1.jpg"
		to   := "/path/to/dir2/file2.jpg"

	  result := path.RelativeFromTo(from, to)
	  log.Println(result)

	  -> "../dir2/file2.jpg"
*/

func RelativeFromTo(from_file, to_file string) (string, error) {
	// Clean and normalize the paths
	absFrom := filepath.Clean(from_file)
	absTo := filepath.Clean(to_file)

	// Split the paths into slices
	fromParts := strings.Split(filepath.ToSlash(absFrom), "/")
	toParts := strings.Split(filepath.ToSlash(absTo), "/")

	// Find the common prefix
	i := 0
	for i < len(fromParts) && i < len(toParts) && fromParts[i] == toParts[i] {
		i++
	}

	// If both paths are the same (files are in the same directory)
	if i == len(fromParts)-1 && i == len(toParts)-1 {
		return "./" + toParts[len(toParts)-1], nil
	}

	// Calculate the number of "../" needed to go from "from" to the common ancestor
	relativeParts := make([]string, len(fromParts)-(i+1))
	for j := 0; j < len(relativeParts); j++ {
		relativeParts[j] = ".."
	}

	// Append the remaining parts of the "to" path
	relativeParts = append(relativeParts, toParts[i:]...)

	// Join the parts to form the relative path
	relativePath := strings.Join(relativeParts, "/")

	return relativePath, nil
}

func xRelativeFromTo(from string, to string) string {

	farr := strings.Split(from, "/")
	tarr := strings.Split(to, "/")
	p := 0

	var csb, rpsb string

	for i, fs := range farr {
		ts := tarr[i]
		if fs == ts {
			if i > 0 {
				csb = filepath.Join(csb, fs)
			} else {
				csb += fs
			}
		} else {
			p = len(farr) - (i + 1)
			break
		}
	}

	for i := 0; i < p; i++ {
		if i > 0 {
			rpsb = rpsb + "/.."
		} else {
			rpsb += ".."
		}
	}

	return strings.Replace(to, csb, rpsb, 1)
}
