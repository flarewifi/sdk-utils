/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkslices

func Filter[T any](collection []T, filterFn func(item T) bool) []T {
	var newArr []T = []T{}
	for _, a := range collection {
		ok := filterFn(a)
		if ok {
			newArr = append(newArr, a)
		}
	}
	return newArr
}
