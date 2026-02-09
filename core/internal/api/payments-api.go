package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"core/db/models"
	"core/internal/web/helpers"
	"core/internal/web/middlewares"
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
				DeviceID:       clnt.ID(),
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
	purMw := middlewares.PendingPurchase(self.api.CoreAPI, self.api.models)
	purMw(http.HandlerFunc(handler)).ServeHTTP(w, r)
}

func (self *PaymentsApi) GetPurchaseRequest(r *http.Request) (sdkapi.IPurchaseRequest, error) {
	mdls := self.api.models
	clnt, err := self.api.HttpAPI.GetClientDevice(r)
	if err != nil {
		log.Println("helpers.CurrentClient error:", err)
		return nil, err
	}

	p, err := mdls.Purchase().PendingPurchase(r.Context(), clnt.ID())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("No pending purchase found for device:", clnt.ID())
			return nil, errors.New("No pending purchase found")
		}
		log.Println("mdls.Purchase().FindByDeviceId error:", err)
		return nil, err
	}

	if p.IsCancelled() || p.IsConfirmed() {
		log.Println("Purchase is already processed")
		return nil, errors.New("Purchase is already processed")
	}

	purchase := NewPurchase(self.api, r.Context(), clnt.ID(), p)
	return purchase, nil
}

func (self *PaymentsApi) FindPurchaseRequestByUUID(uuid string) (sdkapi.IPurchaseRequest, error) {
	ctx := context.Background()
	mdls := self.api.models

	p, err := mdls.Purchase().FindByUUID(ctx, uuid)
	if err != nil {
		log.Printf("mdls.Purchase().FindByUUID error for uuid %s: %v", uuid, err)
		return nil, err
	}

	purchase := NewPurchase(self.api, ctx, p.DeviceID(), p)
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

func (self *PaymentsApi) ExtractPurchaseData(r *http.Request) (sdkapi.IPurchaseRequest, error) {
	// Get the purchase token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		return nil, errors.New("missing token query parameter")
	}

	// Verify the token and extract claims
	claims, err := helpers.VerifyPurchaseToken(token)
	if err != nil {
		log.Printf("ExtractPurchaseData: Token verification failed: %v", err)
		return nil, err
	}

	// Find the purchase by UUID
	return self.FindPurchaseRequestByUUID(claims.PurchaseUID)
}

func (self *PaymentsApi) OnPurchaseEvent(event sdkapi.PurchaseEvent, callback func(data sdkapi.PurchaseEventData) error) {
	self.paymentsMgr.OnPurchaseEvent(event, callback)
}
