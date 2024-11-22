/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkstr

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func Sha1Hash(texts ...string) string {
	allstr := strings.Join(texts, "")
	hash := sha1.Sum([]byte(allstr))
	return hex.EncodeToString(hash[:])
}
