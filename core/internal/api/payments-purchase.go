package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"core/db/models"
	sdkapi "sdk/api"
)

func NewPurchase(api *PluginApi, ctx context.Context, deviceId int64, p *models.Purchase) *Purchase {
	return &Purchase{
		api:      api,
		deviceId: deviceId,
		purchase: p,
	}
}

type Purchase struct {
	api      *PluginApi
	deviceId int64
	purchase *models.Purchase
}

func (self *Purchase) Id() int64 {
	return self.purchase.Id()
}

func (self *Purchase) Uid() string {
	return self.purchase.Uid()
}

func (self *Purchase) Sku() string {
	return self.purchase.Sku()
}

func (self *Purchase) Name() string {
	return self.purchase.Name()
}

func (self *Purchase) IsFixedPrice() bool {
	_, isfixed := self.purchase.FixedPrice()
	return isfixed
}

func (self *Purchase) Price() float64 {
	price, _ := self.purchase.FixedPrice()
	return price
}

func (self *Purchase) CreatePayment(tx *sql.Tx, ctx context.Context, params sdkapi.CreatePaymentParams) error {
	mdls := self.api.models
	_, err := mdls.Payment().Create(tx, ctx, models.CreatePaymentParams{
		PurchaseID:    self.purchase.Id(),
		Amount:        params.Amount,
		PaymentMethod: params.Optname,
	})
	return err
}

func (self *Purchase) PayWithWallet(tx *sql.Tx, ctx context.Context, dbt float64) error {
	err := self.purchase.Update(tx, ctx, dbt, nil, self.purchase.CancelledAt(), self.purchase.ConfirmedAt(), nil)
	return err
}

func (self *Purchase) State(tx *sql.Tx, ctx context.Context) (sdkapi.PurchaseState, error) {
	state := sdkapi.PurchaseState{}

	device, err := self.api.models.Device().Find(tx, ctx, self.deviceId)
	if err != nil {
		return state, err
	}

	wallet, err := device.Wallet(tx, ctx)
	if err != nil {
		return state, err
	}

	total, err := self.purchase.TotalPayment(tx, ctx)
	if err != nil {
		return state, err
	}

	walletDebit := self.purchase.WalletDebit()
	walletEndBal := wallet.Balance() - walletDebit

	state.PurchaseID = self.purchase.Id()
	state.TotalPayment = total
	state.WalletDebit = walletDebit
	state.WalletEndingBal = walletEndBal
	state.WalletRealBal = wallet.Balance()

	return state, nil
}

func (self *Purchase) Execute(w http.ResponseWriter, r *http.Request) {
	pmgr := self.api.PluginsMgr()
	callbackPkg, ok := pmgr.FindByPkg(self.purchase.CallbackPluginPkg())
	if !ok {
		self.ErrorPage(w, errors.New("Unable to find plugin to receive the payment."))
		return
	}

	fmt.Println("CallbackPkg: ", callbackPkg)
	callbackPkg.Http().Response().Redirect(w, r, self.purchase.CallbackRoute())
}

func (self *Purchase) Confirm(tx *sql.Tx, ctx context.Context) error {
	return self.purchase.Confirm(tx, ctx)
}

func (self *Purchase) Cancel(tx *sql.Tx, ctx context.Context) error {
	return self.purchase.Cancel(tx, ctx)
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
