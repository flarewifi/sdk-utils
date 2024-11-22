/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkstr

import (
	"regexp"
	"strings"
)

func Slugify(input string, separator string) string {
	if separator == "" {
		separator = "_"
	}

	// Convert to lowercase
	result := strings.ToLower(input)

	// Remove special characters
	re := regexp.MustCompile("[^a-z0-9]+")
	result = re.ReplaceAllString(result, separator)

	// Remove leading and trailing hyphens
	result = strings.Trim(result, separator)

	return result
}
