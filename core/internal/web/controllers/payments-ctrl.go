package controllers

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	paymentsview "core/resources/views/portal/payments"
)

func PaymentOptionsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		result := g.PaymentsMgr.AllOptions(r)
		opts := make([]paymentsview.PaymentOption, len(result))

		for i, opt := range result {
			opts[i] = paymentsview.PaymentOption{
				Name: opt.Name(),
				URL:  opt.URL(),
			}
		}

		paymentsPage := paymentsview.PaymentOptions(opts)
		res.PortalView(w, r, sdkapi.ViewPage{
			PageContent: paymentsPage,
		})
	}
}
