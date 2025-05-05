/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type FormListFieldOption struct {
	Label string
	Value interface{}
}

type FormListOptionType string

const (
	FormListOptionSelect   FormListOptionType = "select"
	FormListOptionRadio    FormListOptionType = "radio"
	FormListOptionCheckbox FormListOptionType = "checkbox"
)

type FormListField struct {
	Name       string
	Label      string
	Type       string
	OptionType FormListOptionType
	Required   bool
	Minimum    int
	Maximum    int
	Multiple   bool
	Options    func() []FormListFieldOption
	ValueFn    func() interface{}
}

func (f FormListField) GetName() string {
	return f.Name
}

func (f FormListField) GetLabel() string {
	return f.Label
}

func (f FormListField) GetType() string {
	return FormFieldTypeList
}

func (f FormListField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return nil
}

func (f FormListField) IsRequired() bool {
	return f.Required
}
