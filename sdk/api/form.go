/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "errors"

const (
	FormFieldTypeBoolean string = "bool"
	FormFieldTypeDecimal string = "decimal"
	FormFieldTypeInteger string = "integer"
	FormFieldTypeList    string = "list"
	FormFieldTypeMulti   string = "multi"
	FormFieldTypeString  string = "string"
	FormFieldTypeText    string = "text"
)

var ErrFormParse = errors.New("parsing error")

type IFormField interface {
	GetName() string
	GetLabel() string
	GetType() string
	GetValue() interface{}
}

type IHttpForm interface {
	GetSection(section string) (sec IFormSection, ok bool)
	GetSections() []IFormSection

	GetStringValue(section string, name string) (string, error)
	GetStringValues(section string, name string) ([]string, error)

	GetIntValue(section string, name string) (int64, error)
	GetIntValues(section string, name string) ([]int64, error)

	GetFloatValue(section string, name string) (float64, error)
	GetFloatValues(section string, name string) ([]float64, error)

	GetBoolValue(section string, name string) (bool, error)
	GetBoolValues(section string, name string) ([]bool, error)

	GetMultiField(section string, name string) (IFormMultiField, error)
}

type HttpForm struct {
	CallbackRoute string
	SubmitLabel   string
	Sections      []FormSection
}

type FormSection struct {
	Name   string
	Label  string
	Fields []IFormField
}

type SectionData struct {
	Name   string
	Fields []FormFieldData
	Errors []string
}

type FormFieldData struct {
	Name  string
	Value interface{}
}
