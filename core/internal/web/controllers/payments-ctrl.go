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
		purchase, err := g.CoreAPI.PaymentsAPI.GetPurchaseRequest(r)
		if err != nil {
			res.RedirectToPortal(w, r)
			return
		}
		result := g.PaymentsMgr.AllOptions(r)
		opts := make([]paymentsview.PaymentOption, len(result))

		for i, opt := range result {
			opts[i] = paymentsview.PaymentOption{
				Label: opt.Label(),
				URL:   opt.URL(),
			}
		}

		paymentsPage := paymentsview.PaymentOptions(g.CoreAPI, purchase, opts)
		res.PortalView(w, r, sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				JsFile:  "payment-options.js",
				CssFile: "payment-options.css",
			},
			PageContent: paymentsPage,
		})
	}
}
