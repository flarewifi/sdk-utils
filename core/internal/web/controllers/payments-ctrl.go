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

func PaymentOptionsListCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := g.PaymentsMgr.AllOptions(r)
		opts := make([]paymentsview.PaymentOption, len(result))

		for i, opt := range result {
			opts[i] = paymentsview.PaymentOption{
				Label: opt.Label(),
				URL:   opt.URL(),
			}
		}

		view := paymentsview.PaymentOptionCard(g.CoreAPI, opts)
		view.Render(r.Context(), w)
	}
}

func CancelPurchaseCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		ctx := r.Context()

		// Get the pending purchase from session
		purchase, err := g.CoreAPI.PaymentsAPI.GetPurchaseRequest(r)
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "No pending purchase found"), sdkapi.FlashMsgError)
			res.RedirectToPortal(w, r)
			return
		}

		// Cancel the purchase (handles refunds automatically)
		err = purchase.Cancel(ctx)
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to cancel purchase"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "payments:options")
			return
		}

		// Show warning message and redirect to portal
		res.FlashMsg(w, r, g.CoreAPI.Translate("warning", "The purchase has been cancelled"), sdkapi.FlashMsgWarning)
		res.RedirectToPortal(w, r)
	}
}
