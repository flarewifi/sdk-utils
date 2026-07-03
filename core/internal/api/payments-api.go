package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"core/db/models"
	"core/db/queries"
	"core/internal/web/helpers"
	"core/internal/web/middlewares"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
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
	self.paymentsMgr.NewPaymentProvider(self.api, provider)
}

func (self *PaymentsApi) Checkout(w http.ResponseWriter, r *http.Request, p sdkapi.PurchaseRequest) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		clnt, err := self.api.Http().GetClientDevice(r)
		if err != nil {
			self.ErrorPage(w, err)
			return
		}

		deviceID := clnt.ID()

		// Give subscribers a chance to veto before the purchase request is created.
		if err := self.emitPurchaseBeforeRequest(ctx, &deviceID, p.Sku, p.Name, p.Description, p.Price, p.AnyPrice, p.Metadata); err != nil {
			self.ErrorPage(w, err)
			return
		}

		_, err = self.api.models.Purchase().Create(
			ctx,
			models.CreatePurchaseParams{
				DeviceID:       &deviceID,
				SKU:            p.Sku,
				Name:           p.Name,
				Description:    p.Description,
				Price:          p.Price,
				AnyPrice:       p.AnyPrice,
				CallbackPlugin: self.api.Info().Package,
				CallbackRoute:  p.CallbackRoute,
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
		return nil, err
	}

	p, err := mdls.Purchase().PendingPurchase(r.Context(), clnt.ID())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("No pending purchase found")
		}
		return nil, err
	}

	if p.IsCancelled() || p.IsConfirmed() {
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
		return nil, err
	}

	// Find the purchase by UUID
	return self.FindPurchaseRequestByUUID(claims.PurchaseUID)
}

func (self *PaymentsApi) OnPurchaseEvent(event sdkapi.PurchaseEvent, callback func(context.Context, sdkapi.PurchaseEventData) error) {
	self.api.EventsMgr.OnPurchaseEvent(event, callback)
}

// HandlePurchaseExecute registers the in-process handler invoked when a payment
// provider calls Purchase.Execute() for any purchase whose callback plugin is
// this plugin.
func (self *PaymentsApi) HandlePurchaseExecute(handler sdkapi.PurchaseExecuteHandler) {
	self.paymentsMgr.registerExecuteHandler(self.api.Info().Package, handler)
}

// CreatePurchase creates a purchase record programmatically without HTTP checkout flow.
// Used for admin-generated purchases like voucher batch sales where no customer device is involved.
func (self *PaymentsApi) CreatePurchase(ctx context.Context, params sdkapi.CreatePurchaseParams) (sdkapi.IPurchaseRequest, error) {
	mdls := self.api.models

	// Give subscribers a chance to veto before the purchase request is created.
	if err := self.emitPurchaseBeforeRequest(ctx, params.DeviceID, params.Sku, params.Name, params.Description, params.Price, false, params.Metadata); err != nil {
		return nil, err
	}

	p, err := mdls.Purchase().Create(ctx, models.CreatePurchaseParams{
		DeviceID:       params.DeviceID,
		SKU:            params.Sku,
		Name:           params.Name,
		Description:    params.Description,
		Price:          params.Price,
		AnyPrice:       false,
		CallbackPlugin: self.api.Info().Package,
		CallbackRoute:  "",
		Metadata:       params.Metadata,
		Processing:     false,
		PaymentUrl:     "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase: %w", err)
	}

	// For admin purchases without device, use device ID 0
	deviceID := int64(0)
	if params.DeviceID != nil {
		deviceID = *params.DeviceID
	}

	purchase := NewPurchase(self.api, ctx, deviceID, p)
	return purchase, nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// emitPurchaseBeforeRequest builds an in-memory preview purchase (ID == 0) from the
// pending request's fields and fires EventPurchaseBeforeRequest. A non-nil error from any
// subscriber cancels the request before the row is created, so no rollback is needed.
// The preview carries the same getters (SKU/name/price/metadata) as a real request so
// subscribers can make an admission decision; the owning device is resolved when known.
func (self *PaymentsApi) emitPurchaseBeforeRequest(ctx context.Context, deviceID *int64, sku, name, description string, price float64, anyPrice bool, metadata map[string]string) error {
	metaJSON := ""
	if len(metadata) > 0 {
		if b, err := json.Marshal(metadata); err == nil {
			metaJSON = string(b)
		}
	}

	var devNull sql.NullInt64
	if deviceID != nil {
		devNull = sql.NullInt64{Int64: *deviceID, Valid: true}
	}

	previewModel, err := models.NewPurchase(self.api.db, self.api.models, &queries.Purchase{
		DeviceID:       devNull,
		Sku:            sku,
		Name:           name,
		Description:    description,
		Price:          price,
		AnyPrice:       anyPrice,
		CallbackPlugin: self.api.Info().Package,
		Metadata:       metaJSON,
	})
	if err != nil {
		return err
	}

	dev := int64(0)
	if deviceID != nil {
		dev = *deviceID
	}
	preview := NewPurchase(self.api, ctx, dev, previewModel)

	// Resolve the owning device if we have one; best-effort (nil for admin purchases).
	var device sdkapi.IClientDevice
	if deviceID != nil {
		if d, derr := self.api.SessionsMgr().FindClientById(ctx, *deviceID); derr == nil {
			device = d
		}
	}

	return self.api.EventsMgr.EmitPurchaseEvent(ctx, sdkapi.EventPurchaseBeforeRequest, sdkapi.PurchaseEventData{
		Purchase: preview,
		Device:   device,
	})
}
