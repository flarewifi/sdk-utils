package middlewares

import (
	"database/sql"
	"errors"
	"net/http"

	"core/db/models"
	sdkapi "sdk/api"
)

// PendingPurchase checks if the user has a pending purchase.
// If yes, redirects to payment options page.
func PendingPurchase(api sdkapi.IPluginApi, mdls *models.Models) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			res := api.Http().Response()

			client, err := api.Http().GetClientDevice(r)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Client device not registered"), sdkapi.FlashMsgError)
				res.RedirectToPortal(w, r)
				return
			}

			purchase, err := mdls.Purchase().PendingPurchase(ctx, client.ID())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				res.FlashMsg(w, r, api.Translate("error", "Client device not registered"), sdkapi.FlashMsgError)
				res.RedirectToPortal(w, r)
				return
			}

			if purchase != nil {
				// If purchase is processing and has a payment URL, redirect to it
				if purchase.Processing() && purchase.PaymentUrl() != "" {
					http.Redirect(w, r, purchase.PaymentUrl(), http.StatusSeeOther)
					return
				}

				// Otherwise, redirect to payment options page with info message
				res.FlashMsg(w, r, api.Translate("info", "You have a pending purchase. Please complete it before proceeding")+".", sdkapi.FlashMsgInfo)
				res.Redirect(w, r, "payments:options")
				return
			}

			next.ServeHTTP(w, r)
		})

		return handler
	}
}
