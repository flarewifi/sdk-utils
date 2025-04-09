/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormIntegerField struct {
	Name     string
	Label    string
	Required bool
	Minimum  int
	Maximum  int
	ValueFn  func() int64
}

func (f FormIntegerField) GetName() string {
	return f.Name
}

func (f FormIntegerField) GetLabel() string {
	return f.Label
}

func (f FormIntegerField) GetType() string {
	return FormFieldTypeInteger
}

func (f FormIntegerField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return 0
}

func (f FormIntegerField) IsRequired() bool {
	return f.Required
}
