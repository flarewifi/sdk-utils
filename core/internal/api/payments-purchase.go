package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"core/db/models"
	"core/internal/web/helpers"
	"core/utils/env"
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
	// Derive provider info from the calling plugin's API instance
	providerPkg := self.api.info.Package
	providerName := self.api.info.Name
	_, err := mdls.Payment().Create(ctx, models.CreatePaymentParams{
		PurchaseID:        self.purchase.ID(),
		Amount:            params.Amount,
		PaymentOptionUUID: params.PaymentOptionUUID,
		ProviderPkg:       providerPkg,
		ProviderName:      providerName,
	})
	return err
}

func (self *Purchase) PayWithWallet(ctx context.Context, dbt float64) error {
	err := self.purchase.Update(ctx, dbt, nil, self.purchase.CancelledAt(), self.purchase.ConfirmedAt(), nil)
	return err
}

func (self *Purchase) State(ctx context.Context) (sdkapi.PurchaseState, error) {
	state := sdkapi.PurchaseState{}

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

	walletDebit := self.purchase.WalletDebit()
	walletEndBal := wallet.Balance() - walletDebit

	state.PurchaseID = self.purchase.ID()
	state.TotalPayment = total
	state.WalletDebit = walletDebit
	state.WalletEndingBal = walletEndBal
	state.WalletRealBal = wallet.Balance()

	return state, nil
}

func (self *Purchase) Execute(ctx context.Context, params sdkapi.ExecuteParams) error {
	pmgr := self.api.PluginsMgr()
	callbackPkg, ok := pmgr.FindByPkg(self.purchase.CallbackPluginPkg())
	if !ok {
		err := errors.New("Unable to find plugin to receive the payment")
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	// Build the webhook URL
	webhookRoute := self.purchase.WebHookRoute()
	if webhookRoute == "" {
		err := errors.New("WebHookRoute is not configured for this purchase")
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, err.Error())
		return err
	}

	webhookURL := callbackPkg.Http().Helpers().UrlForRoute(webhookRoute)

	// Create JWT token with device ID and purchase UUID
	token, err := helpers.CreatePurchaseToken(self.deviceId, self.purchase.UUID())
	if err != nil {
		execErr := fmt.Errorf("failed to create purchase token: %w", err)
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, execErr.Error())
		return execErr
	}

	// Append token as query parameter to webhook URL
	fullURL := env.LocalBaseURL + webhookURL
	if strings.Contains(fullURL, "?") {
		fullURL = fullURL + "&token=" + token
	} else {
		fullURL = fullURL + "?token=" + token
	}

	fmt.Println("Webhook URL:", fullURL)

	// Marshal params to JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		execErr := fmt.Errorf("failed to marshal execute params: %w", err)
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, execErr.Error())
		return execErr
	}

	// Create request with context and JSON body
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		execErr := fmt.Errorf("failed to create webhook request: %w", err)
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, execErr.Error())
		return execErr
	}

	// Set Content-Type header
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("Webhook request created with purchase token")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		execErr := fmt.Errorf("webhook request failed: %w", err)
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, execErr.Error())
		return execErr
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		execErr := fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
		self.emitPurchaseEvent(ctx, sdkapi.EventPurchaseFailed, execErr.Error())
		return execErr
	}

	fmt.Println("Webhook executed successfully")
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
	fmt.Println("Redirecting to callback route:", callbackRoute)

	// Create JWT token containing device ID and purchase UUID
	token, err := helpers.CreatePurchaseToken(self.deviceId, self.purchase.UUID())
	if err != nil {
		fmt.Println("RedirectToCallback: Failed to create purchase token:", err)
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

	fmt.Println("Redirecting to callback URL:", callbackURL)

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
		// Log error but don't fail - events are best-effort
		fmt.Printf("Failed to get device for purchase event: %v\n", err)
		return
	}

	data := sdkapi.PurchaseEventData{
		Purchase: self,
		Device:   device,
		Reason:   reason,
	}

	// Emit through the payments manager
	if err := self.api.PaymentsAPI.paymentsMgr.EmitPurchaseEvent(event, data); err != nil {
		fmt.Printf("Failed to emit purchase event %s: %v\n", event, err)
	}
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
