package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	case sdkapi.FormFileField:
		return validtr.validateFile(w, sec, v, val)
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

							valCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
							valueCookie, valCookieErr := cookie.GetCookie(r, valCookieName)

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
			}

			if _, ok := fld.(sdkapi.FormFileField); ok {
				mainFldErrCookie := fmt.Sprintf("%v_%v%v", sec.GetName(), fld.GetName(), errorCookieName)
				mainFieldErr, _ := cookie.GetCookie(r, mainFldErrCookie)
				if mainFieldErr != "" {
					errorMap[mainFldErrCookie] = mainFieldErr
				}

				// We'll do this loop so that all values from the cookie will be fetched.
				i := 0
				for {
					prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())
					errCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(errorCookieName, prefix), i)
					errCookie, _ := cookie.GetCookie(r, errCookieName)

					valCookieName := fmt.Sprintf("%s%v", fmt.Sprintf(valueCookieName, prefix), i)
					valueCookie, valCookieErr := cookie.GetCookie(r, valCookieName)

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

			prefix := fmt.Sprintf("%s_%s", sec.GetName(), fld.GetName())
			errCookieName := fmt.Sprintf(errorCookieName, prefix)
			valCookieName := fmt.Sprintf(valueCookieName, prefix)

			if valCookie, err := cookie.GetCookie(r, valCookieName); !errors.Is(err, http.ErrNoCookie) {
				valueMap[valCookieName] = valCookie
			}

			if errCookie, err := cookie.GetCookie(r, errCookieName); !errors.Is(err, http.ErrNoCookie) {
				errorMap[errCookieName] = errCookie
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
		errCookieName := fmt.Sprintf("%s%v", errCookieName, i)
		valCookieName := fmt.Sprintf("%s%v", valCookieName, i)

		_, err := cookie.GetCookie(r, valCookieName)
		if errors.Is(err, http.ErrNoCookie) {
			break
		}

		cookie.DeleteCookie(w, errCookieName)
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
	valStr := strings.TrimSpace(fmt.Sprint(val))
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
		errStr = "Must upload a file."
	}

	paths := val.([]string)
	if fld.IsRequired() && len(paths) == 0 {
		errStr = "Must upload a file."
	}

	if fld.MinFiles != 0 && len(paths) < fld.MinFiles {
		errStr = fmt.Sprintf("Must upload atleast %v file(s).", fld.MinFiles)
	}

	if fld.MaxFiles != 0 && len(paths) > fld.MaxFiles {
		errStr = fmt.Sprintf("Must upload at most %v file(s).", fld.MaxFiles)
	}

	// Delete previous error with same cookie name.
	cookie.DeleteCookie(w, errCookieName)
	if errStr != "" {
		cookie.SetCookie(w, errCookieName, errStr)

		return sdkapi.ErrFormParse
	}

	var fileErr error
	for i, path := range paths {
		errCookieName := fmt.Sprintf("%s%v", errCookieName, i)
		valCookieName := fmt.Sprintf("%s%v", valCookieName, i)

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
		fileBaseName := filepath.Base(path)
		if fld.MinSizeMb != 0 && fileSize < fld.MinSizeMb {
			errStr = fmt.Sprintf("%v is too small. The minimum allowed size is %v MB.", fileBaseName, fld.MinSizeMb)
		}

		if fld.MaxSizeMb != 0 && fileSize > fld.MaxSizeMb {
			errStr = fmt.Sprintf("%v is too large. The maximum allowed size is %v MB.", fileBaseName, fld.MaxSizeMb)
		}

		cookie.SetCookie(w, valCookieName, path)

		// Delete previous error with same cookie name.
		cookie.DeleteCookie(w, errCookieName)
		if errStr != "" {
			cookie.SetCookie(w, errCookieName, errStr)
			fileErr = sdkapi.ErrFormParse

			errStr = "" // Set empty for next iteration.

			continue
		}
	}

	return fileErr
}

func bytesToMB(b int64) int {
	bytesInMb := 1_048_576
	return int(b) / bytesInMb
}
