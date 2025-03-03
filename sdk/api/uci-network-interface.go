/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type INetIface struct {
	Section string
	Device  string
	Proto   string
	IpAddr  string
	Netmask string
	Gateway string
}
