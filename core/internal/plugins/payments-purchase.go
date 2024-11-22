package plugins

import (
	"context"
	"errors"
	"net/http"

	"core/internal/db/models"
	sdkpayments "sdk/api/payments"

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

func (self *Purchase) FixedPrice() (float64, bool) {
	return self.purchase.FixedPrice()
}

func (self *Purchase) CreatePayment(amount float64, optname string) error {
	mdls := self.api.models
	_, err := mdls.Payment().Create(self.ctx, self.purchase.Id(), amount, optname)
	return err
}

func (self *Purchase) PayWithWallet(dbt float64) error {
	err := self.purchase.Update(self.ctx, dbt, nil, self.purchase.CancelledAt(), self.purchase.ConfirmedAt(), nil)
	return err
}

func (self *Purchase) State() (sdkpayments.PurchaseState, error) {
	state := sdkpayments.PurchaseState{}

	device, err := self.api.models.Device().Find(self.ctx, self.deviceId)
	if err != nil {
		return state, err
	}

	wallet, err := device.Wallet(self.ctx)
	if err != nil {
		return state, err
	}

	total, err := self.purchase.TotalPayment(self.ctx)
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

func (self *Purchase) Execute(w http.ResponseWriter) {
	pmgr := self.api.PluginsMgr()
	callbackPkg, ok := pmgr.FindByPkg(self.purchase.CallbackPluginPkg())
	if !ok {
		self.ErrorPage(w, errors.New("Unable to find plugin to receive the payment."))
		return
	}

	callbackPkg.Http().HttpResponse().Redirect(w, nil, self.purchase.CallbackVueRouteName())
}

func (self *Purchase) Confirm() error {
	return self.purchase.Confirm(self.ctx)
}

func (self *Purchase) Cancel() error {
	return self.purchase.Cancel(self.ctx)
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
