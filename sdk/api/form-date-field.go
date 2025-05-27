/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

const DateFormat = "2006-01-02"

type FormDateField struct {
	Name     string
	Label    string
	Required bool
	Minimum  string
	Maximum  string
	ValueFn  func() string
}

func (f FormDateField) GetName() string {
	return f.Name
}

func (f FormDateField) GetLabel() string {
	return f.Label
}

func (f FormDateField) GetType() string {
	return FormFieldTypeDate
}

func (f FormDateField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return ""
}

func (f FormDateField) IsRequired() bool {
	return f.Required
}
