/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type IFormMultiField interface {
	NumRows() int
	GetStringValue(row int, name string) (string, error)
	GetIntValue(row int, name string) (int64, error)
	GetFloatValue(row int, name string) (float64, error)
	GetBoolValue(row int, name string) (bool, error)
}

type FormMultiFieldCol struct {
	Name    string
	Label   string
	Type    string
	ValueFn func() interface{}
}

func (col FormMultiFieldCol) GetName() string {
	return col.Name
}

func (col FormMultiFieldCol) GetLabel() string {
	return col.Label
}

func (col FormMultiFieldCol) GetType() string {
	return col.Type
}

func (col FormMultiFieldCol) GetValue() interface{} {
	if col.ValueFn != nil {
		return col.ValueFn()
	}
	return nil
}

type FormMultiField struct {
	Name    string
	Label   string
	Columns func() []FormMultiFieldCol
	ValueFn func() [][]FormFieldData
}

func (f FormMultiField) GetName() string {
	return f.Name
}

func (f FormMultiField) GetLabel() string {
	return f.Label
}

func (f FormMultiField) GetType() string {
	return FormFieldTypeMulti
}

func (f FormMultiField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return [][]FormFieldData{}
}
