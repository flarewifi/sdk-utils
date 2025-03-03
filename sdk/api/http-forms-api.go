/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"

	"github.com/a-h/templ"
)

type IHttpFormsApi interface {
	RegisterForm(name string, factory func(r *http.Request) HttpForm) error
	GetFormTemplate(name string, r *http.Request) (templ.Component, error)
	ParseForm(name string, r *http.Request) (IHttpForm, error)
}
