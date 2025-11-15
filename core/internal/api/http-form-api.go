package api

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"
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
