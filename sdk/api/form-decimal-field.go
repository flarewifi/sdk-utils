/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormDecimalField struct {
	Name      string
	Label     string
	Step      float64
	Precision int
	ValueFn   func() float64
}

func (f FormDecimalField) GetName() string {
	return f.Name
}

func (f FormDecimalField) GetLabel() string {
	return f.Label
}

func (f FormDecimalField) GetType() string {
	return FormFieldTypeDecimal
}

func (f FormDecimalField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return 0.0
}
