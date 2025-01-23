/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkutils

func SliceContains[T comparable](arr []T, item T) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

// TODO: Make this function generic
func SliceReverseString(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func SliceMapString(ts []string, f func(string) string) []string {
	us := make([]string, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func SliceFilter[T any](collection []T, filterFn func(item T) bool) []T {
	var newArr []T = []T{}
	for _, a := range collection {
		ok := filterFn(a)
		if ok {
			newArr = append(newArr, a)
		}
	}
	return newArr
}
