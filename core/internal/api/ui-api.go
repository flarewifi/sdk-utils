package api

import (
	"core/resources/views/bs5utils"
	"log"

	sdkapi "sdk/api"

	"github.com/a-h/templ"
)

func NewUIApi(api *PluginApi) *UIApi {
	return &UIApi{
		api: api,
	}
}

type UIApi struct {
	api *PluginApi
}

func (self *UIApi) Pagination(opts *sdkapi.UIPaginationOpts) templ.Component {
	log.Println("self.api: ", self)
	paginationTemplate := bs5utils.Pagination(self.api, bs5utils.PaginationOpts{
		PageURL:     opts.PageURL,
		PerPage:     opts.PerPage,
		CurrentPage: opts.CurrentPage,
		ItemsCount:  opts.ItemsCount,
		ExtraParams: opts.ExtraParams,
	})

	return paginationTemplate
}
