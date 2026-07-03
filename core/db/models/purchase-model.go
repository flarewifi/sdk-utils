package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarewifi/sdk-utils"
)

type PurchaseModel struct {
	db     *db.Database
	models *Models
}

// CreatePurchaseParams holds parameters for creating a new purchase
type CreatePurchaseParams struct {
	DeviceID       *int64 // Nullable - nil for admin purchases (e.g., voucher batch sales)
	SKU            string
	Name           string
	Description    string
	Price          float64
	AnyPrice       bool
	CallbackPlugin string
	CallbackRoute  string
	Metadata       map[string]string
	Processing     bool
	PaymentUrl     string
}

// UpdatePurchaseParams holds parameters for updating a purchase
type UpdatePurchaseParams struct {
	ID              int64
	CancelledAt     *time.Time
	ConfirmedAt     *time.Time
	CancelledReason *string
	Processing      bool
	PaymentUrl      string
}

func NewPurchaseModel(dtb *db.Database, mdls *Models) *PurchaseModel {
	return &PurchaseModel{dtb, mdls}
}

func (self *PurchaseModel) Create(ctx context.Context, params CreatePurchaseParams) (*Purchase, error) {
	b, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, err
	}

	uid := sdkutils.NewUUID()

	// Convert nullable device ID
	var deviceID sql.NullInt64
	if params.DeviceID != nil {
		deviceID = sql.NullInt64{Int64: *params.DeviceID, Valid: true}
	}

	queryParams := queries.CreatePurchaseParams{
		Uuid:           uid,
		DeviceID:       deviceID,
		Sku:            params.SKU,
		Name:           params.Name,
		Description:    params.Description,
		Price:          params.Price,
		AnyPrice:       params.AnyPrice,
		CallbackPlugin: params.CallbackPlugin,
		CallbackRoute:  params.CallbackRoute,
		Metadata:       string(b),
		Processing:     params.Processing,
		PaymentUrl:     params.PaymentUrl,
		PaymentNote:    "",
	}

	pId, err := self.db.Queries.CreatePurchase(ctx, queryParams)
	if err != nil {
		log.Println("error creating purchase: %w", err)
		return nil, err
	}

	return self.Find(ctx, pId)
}

func (self *PurchaseModel) Find(ctx context.Context, id int64) (*Purchase, error) {
	p, err := self.db.Queries.FindPurchase(ctx, id)
	if err != nil {
		log.Println("error finding purchase: %w", err)
		return nil, err
	}
	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) PendingPurchase(ctx context.Context, deviceId int64) (*Purchase, error) {
	deviceIdParam := sql.NullInt64{Int64: deviceId, Valid: true}
	p, err := self.db.Queries.FindPendingPurchase(ctx, deviceIdParam)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) FindByDeviceId(ctx context.Context, deviceId int64) (*Purchase, error) {
	deviceIdParam := sql.NullInt64{Int64: deviceId, Valid: true}
	p, err := self.db.Queries.FindPurchaseByDeviceId(ctx, deviceIdParam)
	if err != nil {
		log.Printf("error finding purchase by device id %v: %v", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) FindByUUID(ctx context.Context, uuid string) (*Purchase, error) {
	p, err := self.db.Queries.FindPurchaseByUUID(ctx, uuid)
	if err != nil {
		log.Printf("error finding purchase by uuid %v: %v", uuid, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) Update(ctx context.Context, params UpdatePurchaseParams) error {
	var cancellReason string
	if params.CancelledReason != nil {
		cancellReason = *params.CancelledReason
	}

	// Convert *time.Time to interface{} for nullable timestamps
	var cancelledAt interface{}
	if params.CancelledAt != nil {
		cancelledAt = *params.CancelledAt
	}
	var confirmedAt interface{}
	if params.ConfirmedAt != nil {
		confirmedAt = *params.ConfirmedAt
	}

	err := self.db.Queries.UpdatePurchase(ctx, queries.UpdatePurchaseParams{
		CancelledAt:     cancelledAt,
		ConfirmedAt:     confirmedAt,
		CancelledReason: cancellReason,
		Processing:      params.Processing,
		PaymentUrl:      params.PaymentUrl,
		ID:              params.ID,
	})
	if err != nil {
		log.Printf("error updating purchase %v: %v", params.ID, err)
		return err
	}

	return nil
}

// UpdateMetadata updates the metadata field of a purchase
func (self *PurchaseModel) UpdateMetadata(ctx context.Context, purchaseID int64, metadata map[string]string) error {
	b, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("error marshaling metadata: %v", err)
		return err
	}

	err = self.db.Queries.UpdatePurchaseMetadata(ctx, queries.UpdatePurchaseMetadataParams{
		ID:       purchaseID,
		Metadata: string(b),
	})
	if err != nil {
		log.Printf("error updating purchase metadata %v: %v", purchaseID, err)
		return err
	}

	return nil
}

