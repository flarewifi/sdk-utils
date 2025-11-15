package api

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"strconv"
	"strings"
)

func NewHttpFormApi(api *PluginApi) *HttpFormApi {
	return &HttpFormApi{
		api:       api,
		validator: NewHTTPFormValidator(api),
	}
}

type HttpFormApi struct {
	api       *PluginApi
	validator *HTTPFormValidator
}

// FormErrorsImpl implements sdkapi.FormErrors
type FormErrorsImpl struct {
	errors map[string]string
}

// HasError implements sdkapi.FormErrors.HasError
func (fe *FormErrorsImpl) HasError(name string) bool {
	_, ok := fe.errors[name]
	return ok
}

// GetError implements sdkapi.FormErrors.GetError
func (fe *FormErrorsImpl) GetError(name string) string {
	return fe.errors[name]
}

// FormValues implements sdkapi.IFormValues
type FormValues struct {
	values map[string]string
}

// GetStringValue implements sdkapi.IFormValues.GetStringValue
func (fv *FormValues) GetStringValue(name string) (string, error) {
	val, ok := fv.values[name]
	if !ok {
		return "", fmt.Errorf("field %s not found", name)
	}
	return val, nil
}

// GetIntValue implements sdkapi.IFormValues.GetIntValue
func (fv *FormValues) GetIntValue(name string) (int64, error) {
	val, ok := fv.values[name]
	if !ok {
		return 0, fmt.Errorf("field %s not found", name)
	}
	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("field %s is not a valid integer: %w", name, err)
	}
	return intVal, nil
}

// GetFloatValue implements sdkapi.IFormValues.GetFloatValue
func (fv *FormValues) GetFloatValue(name string) (float64, error) {
	val, ok := fv.values[name]
	if !ok {
		return 0, fmt.Errorf("field %s not found", name)
	}
	floatVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("field %s is not a valid float: %w", name, err)
	}
	return floatVal, nil
}

// GetBoolValue implements sdkapi.IFormValues.GetBoolValue
func (fv *FormValues) GetBoolValue(name string) (bool, error) {
	val, ok := fv.values[name]
	if !ok {
		return false, fmt.Errorf("field %s not found", name)
	}
	// Handle common boolean representations
	lowerVal := strings.ToLower(strings.TrimSpace(val))
	switch lowerVal {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("field %s is not a valid boolean: %s", name, val)
	}
}

// GetFilePath implements sdkapi.IFormValues.GetFilePath
func (fv *FormValues) GetFilePath(name string) (string, error) {
	val, ok := fv.values[name]
	if !ok {
		return "", fmt.Errorf("field %s not found", name)
	}
	return val, nil
}

// ParseForm implements sdkapi.IHttpFormsApi.ParseForm
func (self *HttpFormApi) ParseForm(w http.ResponseWriter, r *http.Request, validator sdkapi.FormValidator) (sdkapi.IFormValues, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	formValues, err := self.validator.ValidateAndExtractForm(w, r, validator)
	if err != nil {
		return nil, err
	}

	return formValues, nil
}

// Errors implements sdkapi.IHttpFormsApi.Errors
func (self *HttpFormApi) Errors(w http.ResponseWriter, r *http.Request, validatorName string) sdkapi.FormErrors {
	cookieAPI := self.api.HttpAPI.Cookie()
	errorMap := map[string]string{}

	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, validatorName) {
			fieldName := getFieldName(cookie.Name, validatorName)
			errorMap[fieldName] = cookie.Value

			cookieAPI.DeleteCookie(w, cookie.Name)
		}
	}

	return &FormErrorsImpl{errors: errorMap}
}

func getFieldName(fullText, prefix string) string {
	split := strings.SplitAfter(fullText, fmt.Sprintf("%v-", prefix))

	fieldName := split[len(split)-1]

	return fieldName
}
