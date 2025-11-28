package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"core/db/models"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPaymentsApi(api *PluginApi, pmgr *PaymentsMgr) {
	pmtApi := &PaymentsApi{
		api:         api,
		paymentsMgr: pmgr,
	}
	api.PaymentsAPI = pmtApi
}

type PaymentsApi struct {
	api         *PluginApi
	paymentsMgr *PaymentsMgr
}

func (self *PaymentsApi) NewPaymentProvider(provider sdkapi.IPaymentProvider) {
	log.Println("Registering payment method:", provider.Name())
	self.paymentsMgr.NewPaymentProvider(self.api, provider)
}

func (self *PaymentsApi) Checkout(w http.ResponseWriter, r *http.Request, p sdkapi.PurchaseRequest) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		clnt, err := self.api.Http().GetClientDevice(r)
		if err != nil {
			log.Println("helpers.CurrentClient error:", err)
			self.ErrorPage(w, err)
			return
		}

		_, err = self.api.models.Purchase().Create(
			ctx,
			models.CreatePurchaseParams{
				DeviceID:       clnt.Id(),
				SKU:            p.Sku,
				Name:           p.Name,
				Description:    p.Description,
				Price:          p.Price,
				AnyPrice:       p.AnyPrice,
				CallbackPlugin: self.api.Info().Package,
				CallbackRoute:  p.CallbackRoute,
				WebHookRoute:   p.WebHookRoute,
				Metadata:       p.Metadata,
				Processing:     p.Processing,
				PaymentUrl:     p.PaymentUrl,
			},
		)
		if err != nil {
			self.ErrorPage(w, err)
			return
		}

		coreApi := self.api.CoreAPI
		coreApi.HttpAPI.Response().Redirect(w, r, "payments:options")
	}

	// Prevent createting multiple pending purchases
	purMw := self.api.HttpAPI.middlewares.PendingPurchase()
	purMw(http.HandlerFunc(handler)).ServeHTTP(w, r)
}

func (self *PaymentsApi) GetPurchaseRequest(r *http.Request) (sdkapi.IPurchaseRequest, error) {
	mdls := self.api.models
	clnt, err := self.api.HttpAPI.GetClientDevice(r)
	if err != nil {
		log.Println("helpers.CurrentClient error:", err)
		return nil, err
	}

	p, err := mdls.Purchase().PendingPurchase(r.Context(), clnt.Id())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("No pending purchase found for device:", clnt.Id())
			return nil, errors.New("No pending purchase found")
		}
		log.Println("mdls.Purchase().FindByDeviceId error:", err)
		return nil, err
	}

	if p.IsCancelled() || p.IsConfirmed() {
		log.Println("Purchase is already processed")
		return nil, errors.New("Purchase is already processed")
	}

	purchase := NewPurchase(self.api, r.Context(), clnt.Id(), p)
	return purchase, nil
}

func (self *PaymentsApi) FindPurchaseRequestByUID(uid string) (sdkapi.IPurchaseRequest, error) {
	ctx := context.Background()
	mdls := self.api.models

	p, err := mdls.Purchase().FindByUID(ctx, uid)
	if err != nil {
		log.Printf("mdls.Purchase().FindByUID error for uid %s: %v", uid, err)
		return nil, err
	}

	purchase := NewPurchase(self.api, ctx, p.DeviceId(), p)
	return purchase, nil
}

func (self *PaymentsApi) FormatCurrency(amount float64) string {
	// Get current currency from config
	cfg, err := self.api.ConfigAPI.Application().Get()
	if err != nil {
		// Fallback to USD if config is not available
		return self.formatCurrencyWithCode(amount, "USD")
	}
	return self.formatCurrencyWithCode(amount, cfg.Currency)
}

// formatCurrencyWithCode formats a float64 amount as a currency string with the given currency code.
func (self *PaymentsApi) formatCurrencyWithCode(amount float64, currencyCode string) string {
	// Format with 2 decimal places
	formatted := fmt.Sprintf("%.2f", amount)

	// Get currency symbol from the centralized currency table
	symbol := sdkutils.GetCurrencySymbol(currencyCode)

	// If symbol is the same as currency code (not found), format as "amount code"
	if symbol == currencyCode {
		return formatted + " " + currencyCode
	}

	// Otherwise, use the symbol
	return symbol + formatted
}

func (self *PaymentsApi) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
