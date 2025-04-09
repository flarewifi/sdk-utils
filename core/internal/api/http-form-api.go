package api

import (
	"errors"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"sync"

	"github.com/a-h/templ"
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

func (self *HttpFormApi) RegisterForm(name string, factory func(*http.Request) sdkapi.HttpForm) error {
	if name == "" {
		return errors.New("form name key is required")
	}

	self.forms.Store(name, factory)
	return nil
}

func (self *HttpFormApi) GetFormTemplate(name string, r *http.Request) (templ.Component, error) {
	f, ok := self.forms.Load(name)
	if !ok {
		return nil, fmt.Errorf("Unable to find form with name %s", name)
	}

	factory, ok := f.(func(*http.Request) sdkapi.HttpForm)
	if !ok {
		return nil, fmt.Errorf("Invalid form factory for form %s", name)
	}

	formDef := factory(r)

	httpForm := NewHttpForm(self.api, formDef)

	return httpForm.GetTemplate(r), nil
}

func (self *HttpFormApi) ParseForm(name string, w http.ResponseWriter, r *http.Request) (form sdkapi.IHttpForm, err error) {
	f, ok := self.forms.Load(name)
	if !ok {
		return form, fmt.Errorf("Unable to find form with name %s", name)
	}

	factory, ok := f.(func(*http.Request) sdkapi.HttpForm)
	if !ok {
		return form, fmt.Errorf("Invalid form factory for form %s", name)
	}

	formDef := factory(r)

	httpForm := NewHttpForm(self.api, formDef)

	if err := httpForm.ParseForm(w, r); err != nil {
		return nil, err
	}

	return httpForm, nil
}
