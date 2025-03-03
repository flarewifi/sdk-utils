/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type BrVlanPort struct {
	Device   string
	Tagged   bool
	Untagged bool
	Primary  bool
}

func (p *BrVlanPort) String() string {
	str := p.Device + ":"
	if p.Untagged {
		str += "u"
		if p.Primary {
			str += "*"
		}
	} else if p.Tagged {
		str += "t"
	} else {
		str += "t"
	}

	return str
}
