/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormTextField struct {
	Name    string
	Label   string
	ValueFn func() string
}

func (f FormTextField) GetName() string {
	return f.Name
}

func (f FormTextField) GetLabel() string {
	return f.Label
}

func (f FormTextField) GetType() string {
	return FormFieldTypeText
}

func (f FormTextField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return ""
}
