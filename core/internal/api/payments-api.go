package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	"core/db/models"
	"core/internal/web/helpers"
	sdkapi "sdk/api"
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
		clnt, err := helpers.CurrentClient(r)
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
	clnt, err := helpers.CurrentClient(r)
	if err != nil {
		log.Println("helpers.CurrentClient error:", err)
		return nil, err
	}

	p, err := mdls.Purchase().PendingPurchase(r.Context(), clnt.Id())
	if err != nil {
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

func (self *PaymentsApi) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
