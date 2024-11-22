package plugins

import (
	"errors"
	"fmt"
	sdkforms "sdk/api/forms"
	sdkhttp "sdk/api/http"
	"sync"
)

func NewHttpFormApi(api *PluginApi) *HttpFormApi {
	return &HttpFormApi{
		api:   api,
		forms: sync.Map{},
	}
}

type HttpFormApi struct {
	api   *PluginApi
	forms sync.Map
}

func (self *HttpFormApi) RegisterHttpForms(forms ...sdkforms.Form) error {
	for _, form := range forms {
		if form.Name == "" {
			return errors.New("form name key is required")
		}

		f := NewHttpForm(self.api, form)
		self.forms.Store(form.Name, f)

	}

	return nil
}

func (self *HttpFormApi) GetForm(name string) (form sdkhttp.IHttpForm, err error) {
	f, ok := self.forms.Load(name)
	if !ok {
		return form, fmt.Errorf("http form %s is not registered", name)
	}

	form, ok = f.(sdkhttp.IHttpForm)
	if !ok {
		return form, fmt.Errorf("form %s is not IHttpForm", name)
	}

	return
}
