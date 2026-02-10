package api

import (
	"core/resources/views/ui/bs5"

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
	paginationTemplate := bs5.Pagination(self.api, bs5.PaginationOpts{
		PageURL:     opts.PageURL,
		PerPage:     opts.PerPage,
		CurrentPage: opts.CurrentPage,
		ItemsCount:  opts.ItemsCount,
		ExtraParams: opts.ExtraParams,
	})

	return paginationTemplate
}

func (self *UIApi) LineChart(opts *sdkapi.LineChartOpts) templ.Component {
	return bs5.LineChart(*opts)
}
