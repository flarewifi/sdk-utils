/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkslices

func MapString(ts []string, f func(string) string) []string {
	us := make([]string, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}
