/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type UIPaginationOpts struct {
	PageURL       string
	PerPage       int
	CurrentPage   int
	ItemsCount    int64
	MaxPagerCount int
	ExtraParams   map[string]string
}
