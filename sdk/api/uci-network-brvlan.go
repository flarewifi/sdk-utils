/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "fmt"

type BrVlan struct {
	Device string
	VlanID int
}

func (self *BrVlan) String() string {
	return fmt.Sprintf("%s.%d", self.Device, self.VlanID)
}
