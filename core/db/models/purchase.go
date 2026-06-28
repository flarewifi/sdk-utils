package models

import (
	"context"
	"errors"
	"time"

	"core/db"
	"core/db/queries"

	"github.com/goccy/go-json"
)

func NewPurchase(dtb *db.Database, mdls *Models, p *queries.Purchase) (*Purchase, error) {
	purchase := &Purchase{
		db:     dtb,
		models: mdls,
	}

	if p != nil {
		purchase.id = p.ID
		purchase.uuid = p.Uuid
		if p.DeviceID.Valid {
			purchase.deviceId = &p.DeviceID.Int64
		}
		purchase.sku = p.Sku
		purchase.name = p.Name
		purchase.description = p.Description
		purchase.price = p.Price
		purchase.anyPrice = p.AnyPrice
		purchase.callbackPluginPkg = p.CallbackPlugin
		purchase.callbackRoute = p.CallbackRoute
		purchase.processing = p.Processing
		purchase.paymentUrl = p.PaymentUrl

		metadata := make(map[string]string)
		if len(p.Metadata) > 0 {
			if err := json.Unmarshal([]byte(p.Metadata), &metadata); err != nil {
				return nil, err
			}
		}

		purchase.metadata = metadata
		purchase.cancelledReason = &p.CancelledReason

		// Handle interface{} for nullable timestamps
		if p.ConfirmedAt != nil {
			if confirmedAt, ok := p.ConfirmedAt.(time.Time); ok {
				purchase.confirmedAt = &confirmedAt
			}
		}
		if p.CancelledAt != nil {
			if cancelledAt, ok := p.CancelledAt.(time.Time); ok {
				purchase.cancelledAt = &cancelledAt
			}
		}

		purchase.createdAt = p.CreatedAt
	}

	return purchase, nil
}

type Purchase struct {
	db                *db.Database
	models            *Models
	id                int64
	uuid              string
	deviceId          *int64 // nullable for admin purchases
	sku               string
	name              string
	description       string
	price             float64
	anyPrice          bool
	callbackPluginPkg string
	callbackRoute     string
	metadata          map[string]string
	confirmedAt       *time.Time
	cancelledAt       *time.Time
	cancelledReason   *string
	processing        bool
	paymentUrl        string
	createdAt         time.Time
}

func (self *Purchase) ID() int64 {
	return self.id
}

func (self *Purchase) UUID() string {
	return self.uuid
}

func (self *Purchase) DeviceID() int64 {
	if self.deviceId != nil {
		return *self.deviceId
	}
	return 0
}

func (self *Purchase) Sku() string {
	return self.sku
}

func (self *Purchase) Name() string {
	return self.name
}

func (self *Purchase) Description() string {
	return self.description
}

func (self *Purchase) Price() float64 {
	return self.price
}

func (self *Purchase) AnyPrice() bool {
	return self.anyPrice
}

func (self *Purchase) ConfirmedAt() *time.Time {
	return self.confirmedAt
}

func (self *Purchase) CancelledAt() *time.Time {
	return self.cancelledAt
}

func (self *Purchase) CancelledReason() *string {
	return self.cancelledReason
}

func (self *Purchase) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Purchase) CallbackPluginPkg() string {
	return self.callbackPluginPkg
}

func (self *Purchase) CallbackRoute() string {
	return self.callbackRoute
}

func (self *Purchase) Metadata() map[string]string {
	return self.metadata
}

func (self *Purchase) Processing() bool {
	return self.processing
}

func (self *Purchase) PaymentUrl() string {
	return self.paymentUrl
}

func (self *Purchase) IsConfirmed() bool {
	return self.confirmedAt != nil
}

func (self *Purchase) IsCancelled() bool {
	return self.cancelledAt != nil
}

func (self *Purchase) FixedPrice() (float64, bool) {
	return self.price, !self.anyPrice
}

func (self *Purchase) Device(ctx context.Context) (*Device, error) {
	if self.deviceId == nil {
		return nil, errors.New("no device associated with this purchase")
	}
	return self.models.deviceModel.Find(ctx, *self.deviceId)
}

func (self *Purchase) Confirm(ctx context.Context) error {
	now := time.Now().UTC()

	// Clear processing state when purchase is confirmed.
	// Must be set before Update() so it gets persisted to database.
	self.processing = false
	self.paymentUrl = ""

	return self.Update(ctx, nil, &now, nil)
}

func (self *Purchase) Cancel(ctx context.Context) error {
	reason := "Cancelled purchase: " + self.description
	cancelledAt := time.Now().UTC()

	// Clear processing state when purchase is cancelled.
	// Must be set before Update() so it gets persisted to database.
	self.processing = false
	self.paymentUrl = ""

	return self.Update(ctx, &cancelledAt, nil, &reason)
}

func (self *Purchase) Payments(ctx context.Context) ([]*Payment, error) {
	return self.models.paymentModel.FindAllByPurchase(ctx, self.id)
}

func (self *Purchase) TotalPayment(ctx context.Context) (float64, error) {
	pmts, err := self.Payments(ctx)
	if err != nil {
		return 0, err
	}

	var total float64

	for _, p := range pmts {
		total += p.Amount()
	}

	return total, nil
}

func (self *Purchase) Update(ctx context.Context, cancelledAt, confirmedAt *time.Time, reason *string) error {
	err := self.models.purchaseModel.Update(ctx, UpdatePurchaseParams{
		ID:              self.id,
		CancelledAt:     cancelledAt,
		ConfirmedAt:     confirmedAt,
		CancelledReason: reason,
		Processing:      self.processing,
		PaymentUrl:      self.paymentUrl,
	})
	if err != nil {
		return err
	}

	self.cancelledAt = cancelledAt
	self.confirmedAt = confirmedAt
	self.cancelledReason = reason

	return nil
}

func (self *Purchase) SetProcessing(ctx context.Context, paymentUrl string) error {
	// If paymentUrl is empty, clear processing state
	// If paymentUrl is provided, set processing to true
	processing := paymentUrl != ""

	err := self.models.purchaseModel.Update(ctx, UpdatePurchaseParams{
		ID:              self.id,
		CancelledAt:     self.cancelledAt,
		ConfirmedAt:     self.confirmedAt,
		CancelledReason: self.cancelledReason,
		Processing:      processing,
		PaymentUrl:      paymentUrl,
	})
	if err != nil {
		return err
	}

	self.processing = processing
	self.paymentUrl = paymentUrl

	return nil
}

// UpdateMetadata updates the metadata field of the purchase
func (self *Purchase) UpdateMetadata(ctx context.Context, metadata map[string]string) error {
	err := self.models.purchaseModel.UpdateMetadata(ctx, self.id, metadata)
	if err != nil {
		return err
	}
	self.metadata = metadata
	return nil
}
