/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormStringField struct {
	Name     string
	Label    string
	ReadOnly bool
	Password bool
	ValueFn  func() string
}

func (f FormStringField) GetName() string {
	return f.Name
}

func (f FormStringField) GetLabel() string {
	return f.Label
}

func (f FormStringField) GetType() string {
	return FormFieldTypeString
}

func (f FormStringField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return ""
}

func (f FormStringField) IsReadOnly() bool {
	return f.ReadOnly
}

func (f FormStringField) IsPassword() bool {
	return f.Password
}
