/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormBooleanField struct {
	Name    string
	Label   string
	ValueFn func() bool
}

func (f FormBooleanField) GetName() string {
	return f.Name
}

func (f FormBooleanField) GetLabel() string {
	return f.Label
}

func (f FormBooleanField) GetType() string {
	return FormFieldTypeBoolean
}

func (f FormBooleanField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return false
}
