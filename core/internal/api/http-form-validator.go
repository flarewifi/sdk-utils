package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	sdkapi "sdk/api"
	"strconv"
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

	case sdkapi.FormDateField:
		dateVal := strings.TrimSpace(fmt.Sprint(val))
		if v.IsRequired() && dateVal == "" {
			errStr = validtr.api.Translate("error", "date_required_error", "label", v.GetLabel())
		}
	case sdkapi.FormTextField:
		errStr = validtr.validateString(v.IsRequired(), fmt.Sprint(val), fld.GetLabel(), v.Minimum, v.Maximum)
	case sdkapi.FormStringField:
		errStr = validtr.validateString(v.IsRequired(), fmt.Sprint(val), fld.GetLabel(), v.Minimum, v.Maximum)
	case sdkapi.FormIntegerField:
		errStr = validtr.validateInteger(val, v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormDecimalField:
		errStr = validtr.validateDecimal(val, v.Minimum, v.Maximum, fld.GetLabel())
	case sdkapi.FormListField:
		singleRequired := !v.Multiple && v.IsRequired()
		if val == nil && singleRequired {
			errStr = validtr.api.Translate("error", "list_option_required_error", "label", v.GetLabel())
		}

		if v.Multiple {
			valString, errStr = validtr.validateMultipleList(val, v.Minimum, v.Maximum, v.GetLabel())
		}

	case sdkapi.FormMultiField:
		return validtr.validateMultiFieldForm(w, sec, v, val)

	case sdkapi.FormFileField:
		return validtr.validateFile(w, sec, v, val)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(valString))
	cookie.SetCookie(w, fmt.Sprintf(valueCookieName, prefix), encoded)
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

							valCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
							valueCookie, valCookieErr := cookie.GetCookie(r, valCookieName)

							if decoded, err := base64.StdEncoding.DecodeString(valueCookie); err == nil {
								valueCookie = string(decoded)
							}

							// No more cookie in this scenario.
							if errors.Is(valCookieErr, http.ErrNoCookie) {
								break
							}

							valueMap[valCookieName] = valueCookie
							if errCookie != "" {
								errorMap[errCookieName] = errCookie
							}

							i++
						}
					}
				}

				continue
			}

			if _, ok := fld.(sdkapi.FormFileField); ok {
				prefix := fmt.Sprintf("%v_%v", sec.GetName(), fld.GetName())
				errCookieName := fmt.Sprintf(errorCookieName, prefix)
				if errCookie, err := cookie.GetCookie(r, errCookieName); err == nil {
					if decoded, err := base64.StdEncoding.DecodeString(errCookie); err == nil {
						errCookie = string(decoded)
					}
					errorMap[errCookieName] = errCookie
				}

				// We'll do this loop so that all values from the cookie will be fetched.
				i := 0
				for {
					prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())

					valCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
					valueCookie, valCookieErr := cookie.GetCookie(r, valCookieName)

					// No more cookie in this scenario.
					if errors.Is(valCookieErr, http.ErrNoCookie) {
						break
					}

					if decoded, err := base64.StdEncoding.DecodeString(valueCookie); err == nil {
						valueCookie = string(decoded)
					}

					valueMap[valCookieName] = valueCookie
					i++
				}

				continue
			}

			prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())
			errCookie := fmt.Sprintf(errorCookieName, prefix)
			valueCookie := fmt.Sprintf(valueCookieName, prefix)

			if errStrCookie, err := cookie.GetCookie(r, errCookie); !errors.Is(err, http.ErrNoCookie) {
				errorMap[errCookie] = errStrCookie
			}
			if valCookie, err := cookie.GetCookie(r, valueCookie); !errors.Is(err, http.ErrNoCookie) {
				if decoded, err := base64.StdEncoding.DecodeString(valCookie); err == nil {
					valueMap[valueCookie] = string(decoded)
				}
			}
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

			if ffld, ok := fld.(sdkapi.FormFileField); ok {
				validtr.DeletePreviousFileInputCookies(w, r, sec.GetName(), ffld, 0)
			}

			validtr.api.HttpAPI.httpCookie.DeleteCookie(w, errCookie)
			validtr.api.HttpAPI.httpCookie.DeleteCookie(w, valueCookie)
		}
	}
}

func (validtr *HTTPFormValidator) DeletePreviousFileInputCookies(
	w http.ResponseWriter,
	r *http.Request,
	sectionName string,
	fld sdkapi.FormFileField,
	startIndex int,
) {
	prefix := fmt.Sprintf("%v_%v", sectionName, fld.Name)
	cookie := validtr.api.HttpAPI.httpCookie
	errCookieName := fmt.Sprintf(errorCookieName, prefix)
	valCookieName := fmt.Sprintf(valueCookieName, prefix)

	cookie.DeleteCookie(w, errCookieName)
	cookie.DeleteCookie(w, valCookieName)

	i := startIndex
	for {
		valCookieName := fmt.Sprintf("%s%v", valCookieName, i)

		_, err := cookie.GetCookie(r, valCookieName)
		if errors.Is(err, http.ErrNoCookie) {
			break
		}

		cookie.DeleteCookie(w, valCookieName)

		i++
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
		parseErr = validtr.api.Translate("error", "less_than_minimum_rows_error", "value", mfld.Minimum)
	}

	if mfld.Maximum != 0 && numRows > mfld.Maximum {
		parseErr = validtr.api.Translate("error", "more_than_maximum_rows_error", "value", mfld.Maximum)
	}

	// This is to delete previous error in the cookie.
	cookie.DeleteCookie(w, fmt.Sprintf("%v_%v_error", sec.Name, mfld.Name))
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
						errStr = validtr.validateString(col.Required, fmt.Sprint(data.Value), col.Label, col.Minimum, col.Maximum)

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

			encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(data.Value)))
			cookie.SetCookie(w, fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i), encoded)
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
		errStr = validtr.api.Translate("error", "less_than_minimum_amount_error", "label", label, "value", min-1)
	}

	if max != 0 && valInt > max {
		errStr = validtr.api.Translate("error", "more_than_maximum_amount_error", "label", label, "value", max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateDecimal(val any, min, max int, label string) (errStr string) {
	fval := val.(float64)
	if fval < float64(min) {
		errStr = validtr.api.Translate("error", "less_than_minimum_amount_error", "label", label, "value", min-1)
	}

	if float64(max) != 0 && fval > float64(max) {
		errStr = validtr.api.Translate("error", "more_than_maximum_amount_error", "label", label, "value", max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateString(required bool, val, label string, min, max int) (errStr string) {
	valStr := strings.TrimSpace(fmt.Sprint(val))

	if required && valStr == "" {
		errStr = validtr.api.Translate("error", "empty_string_field_error", "label", label)

		return
	}

	if len(valStr) < min {
		errStr = validtr.api.Translate("error", "less_than_minimum_chars_error", "label", label, "value", min)
	}

	if max != 0 && len(valStr) > max {
		errStr = validtr.api.Translate("error", "more_than_maximum_chars_error", "label", label, "value", max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateMultipleList(val any, min, max int, label string) (valStr, errStr string) {
	valStr = fmt.Sprint(val)

	listVal := reflect.ValueOf(val)
	if listVal.Kind() == reflect.Slice {
		count := listVal.Len()
		if count < min {
			errStr = validtr.api.Translate("error", "less_than_minimum_selection_error", "value", min, "label", label)
		}

		if max != 0 && count > max {
			errStr = validtr.api.Translate("error", "more_than_maximum_selection_error", "value", max, "label", label)
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

func (validtr *HTTPFormValidator) validateFile(
	w http.ResponseWriter,
	sec sdkapi.FormSection,
	fld sdkapi.FormFileField,
	val any,
) error {
	prefix := fmt.Sprintf("%v_%v", sec.Name, fld.Name)
	cookie := validtr.api.HttpAPI.httpCookie
	errCookieName := fmt.Sprintf(errorCookieName, prefix)
	valCookieName := fmt.Sprintf(valueCookieName, prefix)

	var errStr string
	if fld.IsRequired() && val == nil {
		errStr = validtr.api.Translate("error", "file_required_error")
	}

	paths := val.([]string)
	if fld.IsRequired() && len(paths) == 0 {
		errStr = validtr.api.Translate("error", "file_required_error")
	}

	if fld.MinFiles != 0 && len(paths) < fld.MinFiles {
		errStr = validtr.api.Translate("error", "less_than_minimum_file_count_error", "value", fld.MinFiles)
	}

	if fld.MaxFiles != 0 && len(paths) > fld.MaxFiles {
		errStr = validtr.api.Translate("error", "more_than_maximum_file_count_error", "value", fld.MaxFiles)
	}

	// Delete previous error with same cookie name.
	cookie.DeleteCookie(w, errCookieName)
	if errStr != "" {
		cookie.SetCookie(w, errCookieName, errStr)
		return sdkapi.ErrFormParse
	}

	var validCount int
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			log.Printf("%v: Failed to open file %s\n", err, file.Name())
			continue
		}

		stat, err := file.Stat()
		if err != nil {
			log.Printf("%v: Failed to open file %s stat \n", err, file.Name())
			continue
		}

		fileSize := bytesToMB(stat.Size())
		belowMin := fld.MinSizeMb != 0 && fileSize < fld.MinSizeMb
		exceedMax := fld.MaxSizeMb != 0 && fileSize > fld.MaxSizeMb
		if belowMin || exceedMax {
			if err := os.Remove(path); err != nil {
				log.Printf("%v: Failed to remove file %s \n", err, file.Name())
			}
			continue
		}

		valCookieName := fmt.Sprintf("%s%v", valCookieName, validCount)
		cookie.SetCookie(w, valCookieName, path)
		validCount++
	}

	if validCount == 0 && len(paths) > 0 {
		var errMsg strings.Builder

		errMsg.WriteString(fmt.Sprintf("%s\n", validtr.api.Translate("error", "no_file_found_error")))

		if fld.MinSizeMb > 0 {
			errStr := validtr.api.Translate("error", "less_than_minimum_file_size", "value", fld.MinSizeMb)
			errMsg.WriteString(fmt.Sprintf("\t- %v\n", errStr))
		}
		if fld.MaxSizeMb > 0 {
			errStr := validtr.api.Translate("error", "more_than_maximum_file_size", "value", fld.MaxSizeMb)
			errMsg.WriteString(fmt.Sprintf("\t- %v", errStr))
		}

		errStr = base64.StdEncoding.EncodeToString([]byte(errMsg.String()))
		cookie.SetCookie(w, errCookieName, errStr)
		return sdkapi.ErrFormParse
	}

	return nil
}

func bytesToMB(b int64) int {
	bytesInMb := 1_048_576
	return int(b) / bytesInMb
}

func (validtr *HTTPFormValidator) ValidateForm(w http.ResponseWriter, r *http.Request, form sdkapi.FormWithValidator) error {
	var validateErr error
	cookie := validtr.api.HttpAPI.Cookie()

	for _, validator := range form.FormValidators {
		var (
			fieldName  = validator.FieldName
			fieldLabel = validator.FieldLabel
			fieldType  = validator.FieldType
			rules      = validator.FieldRules

			cookieName = fmt.Sprintf("%v-%v", form.FormName, fieldName)

			val = r.FormValue(fieldName)
		)

		if !rules.Required {
			continue
		}

		switch fieldType {
		case sdkapi.FormFieldTypeString, sdkapi.FormFieldTypeText:
			errStr := validtr.validateString(rules.Required, val, fieldLabel, rules.Minimum, rules.Maximum)
			if errStr != "" {
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}
		case sdkapi.FormFieldTypeInteger:
			valInt, err := strconv.Atoi(val)
			if err != nil {
				valInt = 0
			}

			errStr := validtr.validateInteger(valInt, rules.Minimum, rules.Maximum, fieldLabel)
			if errStr != "" {
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}

		case sdkapi.FormFieldTypeDecimal:
			valFloat, err := strconv.ParseFloat(val, 64)
			if err != nil {
				valFloat = 0.0
			}

			errStr := validtr.validateDecimal(valFloat, rules.Minimum, rules.Maximum, fieldLabel)
			if errStr != "" {
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}

		case sdkapi.FormFieldTypeList:
			vals := r.Form[fieldName]
			if val == "" || len(vals) == 0 || (len(vals) == 1 && strings.TrimSpace(vals[0]) == "") {
				errStr := validtr.api.Translate("error", "list_option_required_error", "label", fieldLabel)
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}

			if len(vals) < rules.Minimum {
				errStr := validtr.api.Translate("error", "less_than_minimum_selection_error", "value", rules.Minimum, "label", fieldLabel)
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}

			if rules.Maximum != 0 && len(vals) > rules.Maximum {
				errStr := validtr.api.Translate("error", "more_than_maximum_selection_error", "value", rules.Maximum, "label", fieldLabel)
				cookie.SetCookie(w, cookieName, errStr)
				validateErr = errors.New("field validation error")
				continue
			}

		default:
			return errors.New("no validation checking set for this type")
		}
	}

	if validateErr != nil {
		return errors.New("parsing error")
	}
	return nil
}
