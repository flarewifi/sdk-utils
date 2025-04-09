package api

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	sdkapi "sdk/api"
	"strings"
)

func NewHTTPFormValidator(api *PluginApi) *HTTPFormValidator {
	return &HTTPFormValidator{
		api: api,
	}
}

var errorCookieName = "%v_field_error"
var valueCookieName = "%v_field_value"

type HTTPFormValidator struct {
	api *PluginApi
}

func (validtr *HTTPFormValidator) ValidateFormField(
	w http.ResponseWriter,
	sec sdkapi.FormSection,
	fld sdkapi.IFormField,
	val any,
) error {
	var (
		errStr string

		prefix    = fmt.Sprintf("%s_%s", sec.Name, fld.GetName())
		cookie    = validtr.api.HttpAPI.httpCookie
		valString = fmt.Sprint(val)
	)

	switch v := fld.(type) {
	case sdkapi.FormBooleanField:
		// No validation handling for boolean.
		return nil

	case sdkapi.FormTextField:
		if !v.IsRequired() {
			return nil
		}

		errStr = validtr.validateString(fmt.Sprint(val), v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormStringField:
		if !v.IsRequired() {
			return nil
		}

		errStr = validtr.validateString(fmt.Sprint(val), v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormIntegerField:
		if !v.IsRequired() {
			return nil
		}

		errStr = validtr.validateInteger(val, v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormDecimalField:
		if !v.IsRequired() {
			return nil
		}

		errStr = validtr.validateDecimal(val, v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormListField:
		singleRequired := !v.Multiple && v.IsRequired()
		if val == nil && singleRequired {
			errStr = fmt.Sprintf("Must choose one of the %vs.", v.GetLabel())
		}

		if v.Multiple {
			valString, errStr = validtr.validateMultipleList(val, v.Minimum, v.Maximum, v.GetLabel())
		}

	case sdkapi.FormMultiField:
		return validtr.validateMultiFieldForm(w, sec, v, val)
	}

	cookie.SetCookie(w, fmt.Sprintf(valueCookieName, prefix), valString)
	if errStr != "" {
		cookie.SetCookie(w, fmt.Sprintf(errorCookieName, prefix), errStr)

		return sdkapi.ErrFormParse
	}

	// If the field is valid, we'll remove the previous error from the cookie.
	cookie.DeleteCookie(w, fmt.Sprintf(errorCookieName, prefix))

	return nil
}

func (validtr *HTTPFormValidator) GetValidatedValues(
	r *http.Request,
	form sdkapi.IHttpForm,
) (errorMap, valueMap map[string]string) {
	cookie := validtr.api.HttpAPI.httpCookie

	errorMap = map[string]string{}
	valueMap = map[string]string{}

	for _, sec := range form.GetSections() {
		for _, fld := range sec.GetFields() {
			if mfld, ok := fld.(sdkapi.FormMultiField); ok {
				mainFldErrCookie := fmt.Sprintf("%v_%v_error", sec.GetName(), fld.GetName())
				mainFieldErr, _ := cookie.GetCookie(r, mainFldErrCookie)
				if mainFieldErr != "" {
					errorMap[mainFldErrCookie] = mainFieldErr
				}

				mfldData := mfld.ValueFn()
				for _, sliceFormFieldData := range mfldData {
					for _, formFielData := range sliceFormFieldData {

						// We'll do this loop instead of basing on the mfldData index
						// so that all values from the cookie will be fetched.
						i := 0
						for {
							prefix := fmt.Sprintf("%s_%s_%s", sec.GetName(), fld.GetName(), formFielData.Name)
							errCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(errorCookieName, prefix), i)
							errCookie, _ := cookie.GetCookie(r, errCookieName)

							valueCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
							valueCookie, valCookieErr := cookie.GetCookie(r, valueCookieName)

							// No more cookie in this scenario.
							if errors.Is(valCookieErr, http.ErrNoCookie) {
								break
							}

							errorMap[errCookieName] = errCookie
							valueMap[valueCookieName] = valueCookie

							i++
						}
					}
				}
			}

			prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())
			errCookie := fmt.Sprintf(errorCookieName, prefix)
			valueCookie := fmt.Sprintf(valueCookieName, prefix)

			errorMap[errCookie], _ = cookie.GetCookie(r, errCookie)
			valueMap[valueCookie], _ = cookie.GetCookie(r, valueCookie)
		}
	}

	return errorMap, valueMap
}

func (validtr *HTTPFormValidator) DeleteAllFormCookies(w http.ResponseWriter, r *http.Request, form sdkapi.IHttpForm) {
	for _, sec := range form.GetSections() {
		for _, fld := range sec.GetFields() {
			prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())
			errCookie := fmt.Sprintf(errorCookieName, prefix)
			valueCookie := fmt.Sprintf(valueCookieName, prefix)

			if mfld, ok := fld.(sdkapi.FormMultiField); ok {
				value := mfld.GetValue()
				mfldData, ok := value.([][]sdkapi.FormFieldData)
				if !ok {
					continue
				}

				mainFldErrCookie := fmt.Sprintf("%v_%v_error", sec.GetName(), fld.GetName())
				validtr.api.HttpAPI.httpCookie.DeleteCookie(w, mainFldErrCookie)

				for i, sliceData := range mfldData {
					for _, data := range sliceData {
						prefix = fmt.Sprintf("%s_%s_%s", sec.GetName(), fld.GetName(), data.Name)

						errCookie = fmt.Sprintf("%s%v", fmt.Sprintf(errorCookieName, prefix), i)
						valueCookie = fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
						valueCookie = fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)

						validtr.api.HttpAPI.httpCookie.DeleteCookie(w, errCookie)
						validtr.api.HttpAPI.httpCookie.DeleteCookie(w, valueCookie)
					}
				}

				continue
			}

			validtr.api.HttpAPI.httpCookie.DeleteCookie(w, errCookie)
			validtr.api.HttpAPI.httpCookie.DeleteCookie(w, valueCookie)
		}
	}
}

func (validtr *HTTPFormValidator) validateMultiFieldForm(
	w http.ResponseWriter,
	sec sdkapi.FormSection,
	mfld sdkapi.FormMultiField,
	val any,
) error {
	var (
		parseErr string
		cookie   = validtr.api.HttpAPI.httpCookie
	)

	if !mfld.IsRequired() {
		return nil
	}

	var mfdata [][]sdkapi.FormFieldData
	if val != nil {
		data, ok := val.([][]sdkapi.FormFieldData)
		if !ok {
			return fmt.Errorf("section %s, field %s value is not a slice of sdkapi.FieldData, instead %T", sec, mfld.GetName(), val)
		}

		mfdata = data
	}

	numRows := len(mfdata)
	if numRows < mfld.Minimum {
		parseErr = fmt.Sprintf("Must have at least %v rows.", mfld.Minimum)
	}

	if mfld.Maximum != 0 && numRows > mfld.Maximum {
		parseErr = fmt.Sprintf("Must not exceed  %v rows.", mfld.Maximum)
	}

	if parseErr != "" {
		cookie.SetCookie(w, fmt.Sprintf("%v_%v_error", sec.Name, mfld.Name), parseErr)
	}

	for i, sliceData := range mfdata {
		for _, data := range sliceData {
			prefix := fmt.Sprintf("%v_%v_%v", sec.Name, mfld.Name, data.Name)

			var errStr string
			for _, col := range mfld.Columns() {
				if col.Name == data.Name {
					switch col.GetType() {
					case sdkapi.FormFieldTypeText, sdkapi.FormFieldTypeString:
						errStr = validtr.validateString(fmt.Sprint(data.Value), col.Minimum, col.Maximum, col.Label)

					case sdkapi.FormFieldTypeDecimal:
						errStr = validtr.validateDecimal(data.Value, col.Minimum, col.Maximum, col.Label)

					case sdkapi.FormFieldTypeInteger:
						errStr = validtr.validateInteger(data.Value, col.Minimum, col.Maximum, col.Label)

					case sdkapi.FormFieldTypeBoolean:
						// No validation handling for boolean.
						break
					}

					// Break inner loop since match is found.
					break
				}
			}

			cookie.SetCookie(w, fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i), fmt.Sprint(data.Value))
			if errStr != "" {
				cookie.SetCookie(w, fmt.Sprintf("%s%v", fmt.Sprintf(errorCookieName, prefix), i), errStr)
				parseErr = errStr

				continue
			}

			// If the field is valid, we'll remove the previous error from the cookie.
			cookie.DeleteCookie(w, fmt.Sprintf(fmt.Sprintf(errorCookieName, prefix), i))
		}
	}

	if parseErr != "" {
		return sdkapi.ErrFormParse
	}

	return nil
}

func (validtr *HTTPFormValidator) validateInteger(val any, min, max int, label string) (errStr string) {
	valInt, ok := val.(int)
	if !ok {
		valInt = int(val.(int64))
	}

	if valInt < min {
		errStr = fmt.Sprintf("%v must be more than %v.", label, min-1)
	}

	if max != 0 && valInt > max {
		errStr = fmt.Sprintf("%v must be less than %v.", label, max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateDecimal(val any, min, max int, label string) (errStr string) {
	fval := val.(float64)
	if fval < float64(min) {
		errStr = fmt.Sprintf("%v must be more than %v.", label, min-1)
	}

	if float64(max) != 0 && fval > float64(max) {
		errStr = fmt.Sprintf("%v must be less than %v.", label, max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateString(val string, min, max int, label string) (errStr string) {
	valStr := fmt.Sprint(val)
	if valStr == "" {
		errStr = fmt.Sprintf("%v must not be empty.", label)

		return
	}

	if len(valStr) < min {
		errStr = fmt.Sprintf("%v must have at least %v characters.", label, min)
	}

	if max != 0 && len(valStr) > max {
		errStr = fmt.Sprintf("%v must not exceed %v characters.", label, max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateMultipleList(val any, min, max int, label string) (valStr, errStr string) {
	valStr = fmt.Sprint(val)

	listVal := reflect.ValueOf(val)
	if listVal.Kind() == reflect.Slice {
		count := listVal.Len()
		if count < min {
			errStr = fmt.Sprintf("Must choose at least %v of the %vs.", min, label)
		}

		if max != 0 && count > max {
			errStr = fmt.Sprintf("Must choose at most %v of the %vs.", max, label)
		}

		var sb strings.Builder
		for i := 0; i < listVal.Len(); i++ {
			sb.WriteString(fmt.Sprint(listVal.Index(i).Interface()))
			if i < listVal.Len()-1 {
				sb.WriteString(",")
			}
		}

		valStr = sb.String()
	}

	return valStr, errStr
}
