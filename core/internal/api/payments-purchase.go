package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"core/db/models"
	"core/internal/web/helpers"
	"core/utils/config"
	"core/utils/env"
	sdkapi "sdk/api"

	"github.com/golang-jwt/jwt/v5"
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
	_, err := mdls.Payment().Create(ctx, models.CreatePaymentParams{
		PurchaseID:    self.purchase.ID(),
		Amount:        params.Amount,
		PaymentMethod: params.Optname,
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
		return errors.New("Unable to find plugin to receive the payment")
	}

	// Build the webhook URL
	webhookRoute := self.purchase.WebHookRoute()
	if webhookRoute == "" {
		return errors.New("WebHookRoute is not configured for this purchase")
	}

	webhookURL := callbackPkg.Http().Helpers().UrlForRoute(webhookRoute)

	// Create POST request to local server using env.LocalBaseURL
	fullURL := env.LocalBaseURL + webhookURL

	fmt.Println("Webhook URL:", fullURL)

	// Get application secret for JWT signing
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		return fmt.Errorf("failed to read application config: %w", err)
	}

	// Create JWT token with 1-minute expiration
	now := time.Now()
	claims := helpers.WebhookClaims{
		DeviceID:    self.deviceId,
		PurchaseUID: self.purchase.UUID(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Minute)),
			Issuer:    "flarehotspot-core",
			Subject:   "webhook-auth",
		},
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(appCfg.Secret))
	if err != nil {
		return fmt.Errorf("failed to sign JWT token: %w", err)
	}

	// Set device ID in params
	params.DeviceID = self.deviceId

	// Marshal params to JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal execute params: %w", err)
	}

	// Create request with context and JSON body
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Add JWT token in Authorization header
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Purchase-Webhook", "true")

	fmt.Println("Webhook request created with JWT token")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
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
	callbackPkg.Http().Response().Redirect(w, r, callbackRoute)
}

func (self *Purchase) Confirm(ctx context.Context) error {
	return self.purchase.Confirm(ctx)
}

func (self *Purchase) Cancel(ctx context.Context) error {
	return self.purchase.Cancel(ctx)
}

func (self *Purchase) ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
