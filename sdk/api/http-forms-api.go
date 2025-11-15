/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

type IHttpFormsApi interface {
	ParseFormWithValidator(w http.ResponseWriter, r *http.Request, form FormWithValidator) error
	Errors(w http.ResponseWriter, r *http.Request, formName string) map[string]string
}
