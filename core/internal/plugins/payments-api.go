package plugins

import (
	"errors"
	"log"
	"net/http"

	"core/internal/web/helpers"
	sdkpayments "sdk/api/payments"
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

func (self *PaymentsApi) NewPaymentProvider(provider sdkpayments.IPaymentProvider) {
	log.Println("Registering payment method:", provider.Name())
	self.paymentsMgr.NewPaymentProvider(self.api, provider)
}

func (self *PaymentsApi) Checkout(w http.ResponseWriter, r *http.Request, p sdkpayments.PurchaseRequest) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		clnt, err := helpers.CurrentClient(self.api.ClntReg, r)
		if err != nil {
			log.Println("helpers.CurrentClient error:", err)
			self.ErrorPage(w, err)
			return
		}

		_, err = self.api.models.Purchase().Create(
			r.Context(),
			clnt.Id(),
			p.Sku,
			p.Name,
			p.Description,
			p.Price,
			p.AnyPrice,
			self.api.Pkg(),
			p.CallbackVueRouteName,
		)
		if err != nil {
			log.Println("self.api.models.Purchase().Create error:", err)
			self.ErrorPage(w, err)
			return
		}

		coreApi := self.api.CoreAPI
		coreApi.HttpAPI.HttpResponse().Redirect(w, r, "payments:customer:options")
	}

	purMw := self.api.HttpAPI.middlewares.PendingPurchase()
	purMw(http.HandlerFunc(handler)).ServeHTTP(w, r)
}

func (self *PaymentsApi) GetPendingPurchase(r *http.Request) (sdkpayments.IPurchase, error) {
	mdls := self.api.models
	clnt, err := helpers.CurrentClient(self.api.ClntReg, r)
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

func (self *PaymentsApi) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
