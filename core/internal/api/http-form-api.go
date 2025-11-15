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

func (self *HttpFormApi) ParseFormWithValidator(w http.ResponseWriter, r *http.Request, form sdkapi.FormWithValidator) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	if err := self.validator.ValidateForm(w, r, form); err != nil {
		return err
	}

	return nil
}

func (self *HttpFormApi) Errors(w http.ResponseWriter, r *http.Request, formName string) map[string]string {
	cookieAPI := self.api.HttpAPI.Cookie()
	errorMap := map[string]string{}

	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, formName) {
			fieldName := getFieldName(cookie.Name, formName)
			errorMap[fieldName] = cookie.Value

			cookieAPI.DeleteCookie(w, cookie.Name)
		}
	}

	return errorMap
}

func getFieldName(fullText, prefix string) string {
	split := strings.SplitAfter(fullText, fmt.Sprintf("%v-", prefix))

	fieldName := split[len(split)-1]

	return fieldName
}
