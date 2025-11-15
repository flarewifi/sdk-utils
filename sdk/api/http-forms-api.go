/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

type IHttpFormsApi interface {

	// Parses and validates form data from the http request based on the provided validators.
	ParseForm(w http.ResponseWriter, r *http.Request, validator FormValidator) (IFormValues, error)

	// Retrieves form validation errors
	Errors(w http.ResponseWriter, r *http.Request, validatorName string) FormErrors
}

type IFormValues interface {
	// Returns the input field value as a string
	GetStringValue(name string) (string, error)

	// Returns the input field value as a int64
	GetIntValue(name string) (int64, error)

	// Returns the input field value as a float64
	GetFloatValue(name string) (float64, error)

	// Returns the input field value as a bool
	GetBoolValue(name string) (bool, error)

	// Returns the temp filepath of the uploaded file
	GetFilePath(name string) (string, error)
}

// Form validations -------------

type FormValidator struct {
	Name       string
	Validators []FormFieldValidator
}

type FormFieldValidator struct {
	FieldName  string
	FieldLabel string
	FieldType  FormFieldType
	FieldRules []FormFieldRule
}

type FormFieldType string

const (
	FormFieldTypeBoolean FormFieldType = "bool"
	FormFieldTypeDecimal FormFieldType = "decimal"
	FormFieldTypeInteger FormFieldType = "integer"
	FormFieldTypeString  FormFieldType = "string"
	FormFieldTypeFile    FormFieldType = "file"
)

type FormFieldRule string

const (
	FormFieldRuleRequired FormFieldRule = "required" // is field required
	FormFieldRuleEmail    FormFieldRule = "email"    // is field an email
	FormFieldRuleNumber   FormFieldRule = "number"   // is field a number
	FormFieldRuleMinimum  FormFieldRule = "minimum"  // minimum value or length
	FormFieldRuleMaximum  FormFieldRule = "maximum"  // maximum value or length
	FormFieldRuleFileExt  FormFieldRule = "file_ext" // allowed file extensions separated by comma (if file input)
)

type FormErrors interface {
	// Returns true if there is an error for the given field name
	HasError(name string) bool

	// Returns the error message for the given field name
	GetError(name string) string
}
