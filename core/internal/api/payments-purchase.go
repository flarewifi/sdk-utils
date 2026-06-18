package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"core/db/models"
	"core/internal/web/helpers"
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

func (self *Purchase) ID() int64 {
	return self.purchase.ID()
}

func (self *Purchase) UUID() string {
	return self.purchase.UUID()
}

func (self *Purchase) Sku() string {
	return self.purchase.Sku()
}

func (self *Purchase) WebHookRoute() string {
	return self.purchase.WebHookRoute()
}

func (self *Purchase) Name() string {
	return self.purchase.Name()
}

func (self *Purchase) DeviceID() int64 {
	return self.purchase.DeviceID()
}

func (self *Purchase) Description() string {
	return self.purchase.Description()
}

func (self *Purchase) AnyPrice() bool {
	return self.purchase.AnyPrice()
}

func (self *Purchase) IsFixedPrice() bool {
	_, isfixed := self.purchase.FixedPrice()
	return isfixed
}

func (self *Purchase) Price() float64 {
	price, _ := self.purchase.FixedPrice()
	return price
}

func (self *Purchase) WalletDebit() float64 {
	return self.purchase.WalletDebit()
}

func (self *Purchase) WalletTxID() *int64 {
	return self.purchase.WalletTxID()
}

func (self *Purchase) ConfirmedAt() *time.Time {
	return self.purchase.ConfirmedAt()
}

func (self *Purchase) CancelledAt() *time.Time {
	return self.purchase.CancelledAt()
}

func (self *Purchase) CancelledReason() *string {
	return self.purchase.CancelledReason()
}

func (self *Purchase) CreatedAt() time.Time {
	return self.purchase.CreatedAt()
}

func (self *Purchase) CallbackPluginPkg() string {
	return self.purchase.CallbackPluginPkg()
}

func (self *Purchase) CallbackRoute() string {
	return self.purchase.CallbackRoute()
}

func (self *Purchase) Metadata() map[string]string {
	return self.purchase.Metadata()
}

func (self *Purchase) IsConfirmed() bool {
	return self.purchase.IsConfirmed()
}

func (self *Purchase) IsCancelled() bool {
	return self.purchase.IsCancelled()
}

func (self *Purchase) Processing() bool {
	return self.purchase.Processing()
}

func (self *Purchase) PaymentUrl() string {
	return self.purchase.PaymentUrl()
}

func (self *Purchase) SetProcessing(ctx context.Context, paymentUrl string) error {
	return self.purchase.SetProcessing(ctx, paymentUrl)
}

func (self *Purchase) CreatePayment(ctx context.Context, params sdkapi.CreatePaymentParams) error {
	mdls := self.api.models
	// Derive provider from the calling plugin's package name
	provider := self.api.info.Package
	_, err := mdls.Payment().Create(ctx, models.CreatePaymentParams{
		PurchaseID:        self.purchase.ID(),
		Amount:            params.Amount,
		PaymentOptionUUID: params.ProviderUUID,
		Provider:          provider,
	})
	return err
}

func (self *Purchase) PayWithWallet(ctx context.Context, dbt float64) error {
	err := self.purchase.Update(ctx, dbt, nil, self.purchase.CancelledAt(), self.purchase.ConfirmedAt(), nil)
	return err
}

func (self *Purchase) State(ctx context.Context) (sdkapi.PurchasePaymentData, error) {
	state := sdkapi.PurchasePaymentData{}

	device, err := self.api.models.Device().Find(ctx, self.deviceId)
	if err != nil {
		return state, err
	}

	wallet, err := device.Wallet(ctx)
	if err != nil {
		return state, err
	}

	total, err := self.purchase.TotalPayment(ctx)
	if err != nil {
		return state, err
	}

	// Get first payment's provider (if any)
	payments, err := self.purchase.Payments(ctx)
	if err != nil {
		return state, err
	}
	var paymentProvider string
	if len(payments) > 0 {
		paymentProvider = payments[0].Provider()
	}

	walletDebit := self.purchase.WalletDebit()
	walletEndBal := wallet.Balance() - walletDebit

	state.PurchaseID = self.purchase.ID()
	state.TotalPayment = total
	state.PaymentProvider = paymentProvider
	state.WalletDebit = walletDebit
	state.WalletEndingBal = walletEndBal
	state.WalletRealBal = wallet.Balance()

	return state, nil
}

// Execute dispatches the purchase to the callback plugin's registered
// PurchaseExecuteHandler in-process. This replaces the former loopback HTTP
// "webhook" POST: the handler runs synchronously in the caller's goroutine, so
// there is no second HTTP request, JWT round-trip, TLS, or extra DB connection
// involved — which removes the cross-goroutine contention on the single SQLite
// connection that the HTTP design was prone to.
func (self *Purchase) Execute(ctx context.Context, params sdkapi.ExecuteParams) error {
	pmgr := self.api.PluginsMgr()
	callbackPkg := self.purchase.CallbackPluginPkg()
	if _, ok := pmgr.FindByPkg(callbackPkg); !ok {
		err := errors.New("Unable to find plugin to receive the payment")
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	route := self.purchase.WebHookRoute()
	if route == "" {
		err := errors.New("WebHookRoute is not configured for this purchase")
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	handler, ok := self.api.PaymentsAPI.paymentsMgr.findExecuteHandler(callbackPkg, route)
	if !ok {
		err := errors.New("No payment handler is registered to receive the payment")
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	// Invoke the callback plugin's handler directly. On success the handler is
	// responsible for confirming the purchase (which emits EventPurchaseSuccess);
	// on failure we emit EventPurchaseFailed here so providers see a consistent
	// event regardless of which step failed.
	if err := handler(ctx, self, params); err != nil {
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	return nil
}

func (self *Purchase) RedirectToCallback(w http.ResponseWriter, r *http.Request) {
	pmgr := self.api.PluginsMgr()
	callbackPkg, ok := pmgr.FindByPkg(self.purchase.CallbackPluginPkg())
	if !ok {
		self.ErrorPage(w, errors.New("Unable to find plugin for callback"))
		return
	}

	callbackRoute := self.purchase.CallbackRoute()

	// Create JWT token containing device ID and purchase UUID
	token, err := helpers.CreatePurchaseToken(self.deviceId, self.purchase.UUID())
	if err != nil {
		self.ErrorPage(w, errors.New("failed to create purchase token"))
		return
	}

	// Build callback URL and append token as query parameter
	callbackURL := callbackPkg.Http().Helpers().UrlForRoute(callbackRoute)
	if strings.Contains(callbackURL, "?") {
		callbackURL = callbackURL + "&token=" + token
	} else {
		callbackURL = callbackURL + "?token=" + token
	}

	// Redirect to callback URL with token as query param
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", callbackURL)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, callbackURL, http.StatusSeeOther)
	}
}

func (self *Purchase) Confirm(ctx context.Context) error {
	err := self.purchase.Confirm(ctx)
	if err != nil {
		// Emit failed event on confirmation error
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	// Emit success event
	self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseSuccess, "")
	return nil
}

func (self *Purchase) Cancel(ctx context.Context) error {
	err := self.purchase.Cancel(ctx)
	if err != nil {
		// Emit failed event on cancellation error
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	// Get the cancellation reason from the purchase after Cancel() sets it
	reason := ""
	if self.purchase.CancelledReason() != nil {
		reason = *self.purchase.CancelledReason()
	}

	// Emit cancelled event
	self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseCancelled, reason)
	return nil
}

// emitPurchaseEvent emits a purchase event with the device information.
func (self *Purchase) emitPurchaseEvent(ctx context.Context, event sdkapi.PurchaseEvent, reason string) {
	// Get the device for the event data
	device, err := self.api.SessionsMgr().FindClientById(ctx, self.deviceId)
	if err != nil {
		// Events are best-effort - silently skip on error
		return
	}

	data := sdkapi.PurchaseEventData{
		Purchase: self,
		Device:   device,
		Reason:   reason,
	}

	// Emit through the global EventsManager (async, non-blocking)
	self.api.EventsMgr.EmitPurchaseEvent(context.Background(), event, data)
}

func (self *Purchase) UpdateMetadata(ctx context.Context, metadata map[string]string) error {
	return self.purchase.UpdateMetadata(ctx, metadata)
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
