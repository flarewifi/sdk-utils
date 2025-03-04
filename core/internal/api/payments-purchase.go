package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"core/db/models"
	sdkapi "sdk/api"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func NewPurchase(api *PluginApi, ctx context.Context, deviceId pgtype.UUID, p *models.Purchase) *Purchase {
	return &Purchase{
		api:      api,
		ctx:      ctx,
		deviceId: deviceId,
		purchase: p,
	}
}

type Purchase struct {
	api      *PluginApi
	ctx      context.Context
	deviceId pgtype.UUID
	purchase *models.Purchase
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

func (self *Purchase) CreatePayment(amount float64, optname string) error {
	mdls := self.api.models
	_, err := mdls.Payment().Create(self.ctx, self.purchase.Id(), amount, optname)
	return err
}

func (self *Purchase) PayWithWallet(tx pgx.Tx, dbt float64) error {
	err := self.purchase.Update(tx, self.ctx, dbt, nil, self.purchase.CancelledAt(), self.purchase.ConfirmedAt(), nil)
	return err
}

func (self *Purchase) State(tx pgx.Tx) (sdkapi.PurchaseState, error) {
	state := sdkapi.PurchaseState{}

	device, err := self.api.models.Device().Find(self.ctx, self.deviceId)
	if err != nil {
		return state, err
	}

	wallet, err := device.Wallet(self.ctx)
	if err != nil {
		return state, err
	}

	total, err := self.purchase.TotalPayment(tx, self.ctx)
	if err != nil {
		return state, err
	}

	walletDebit := self.purchase.WalletDebit()
	walletEndBal := wallet.Balance() - walletDebit

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

func (self *Purchase) Confirm(tx pgx.Tx) error {
	return self.purchase.Confirm(tx, self.ctx)
}

func (self *Purchase) Cancel(tx pgx.Tx, ctx context.Context) error {
	return self.purchase.Cancel(tx, ctx)
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
