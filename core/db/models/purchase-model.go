package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	"github.com/google/uuid"
)

type PurchaseModel struct {
	db     *db.Database
	models *Models
}

// CreatePurchaseParams holds parameters for creating a new purchase
type CreatePurchaseParams struct {
	DeviceID       int64
	SKU            string
	Name           string
	Description    string
	Price          float64
	AnyPrice       bool
	CallbackPlugin string
	CallbackRoute  string
	WebHookRoute   string
	Metadata       map[string]string
	Processing     bool
	PaymentUrl     string
}

// UpdatePurchaseParams holds parameters for updating a purchase
type UpdatePurchaseParams struct {
	ID              int64
	WalletDebit     float64
	WalletTxID      *int64
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

	uid := uuid.New()

	queryParams := queries.CreatePurchaseParams{
		Uuid:           uid.String(),
		DeviceID:       params.DeviceID,
		Sku:            params.SKU,
		Name:           params.Name,
		Description:    params.Description,
		Price:          params.Price,
		AnyPrice:       params.AnyPrice,
		CallbackPlugin: params.CallbackPlugin,
		CallbackRoute:  params.CallbackRoute,
		WebhookRoute:   params.WebHookRoute,
		Metadata:       string(b),
		Processing:     params.Processing,
		PaymentUrl:     params.PaymentUrl,
	}

	fmt.Printf("Create Purchase: %+v\n", queryParams)

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
	p, err := self.db.Queries.FindPendingPurchase(ctx, deviceId)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) FindByDeviceId(ctx context.Context, deviceId int64) (*Purchase, error) {
	p, err := self.db.Queries.FindPurchaseByDeviceId(ctx, deviceId)
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

	var walletTxID sql.NullInt64
	if params.WalletTxID != nil {
		walletTxID = sql.NullInt64{Int64: *params.WalletTxID, Valid: true}
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
		WalletDebit:     params.WalletDebit,
		WalletTxID:      walletTxID,
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

// FindByPaymentOptionUUID finds all purchases made with a specific payment option UUID
func (self *PurchaseModel) FindByPaymentOptionUUID(ctx context.Context, paymentOptionUUID string) ([]*Purchase, error) {
	rows, err := self.db.Queries.FindPurchasesByPaymentOptionUUID(ctx, paymentOptionUUID)
	if err != nil {
		log.Printf("error finding purchases by payment option uuid %v: %v", paymentOptionUUID, err)
		return nil, err
	}

	purchases := make([]*Purchase, len(rows))
	for i, row := range rows {
		purchases[i], err = NewPurchase(self.db, self.models, &row)
		if err != nil {
			log.Printf("error creating purchase from row: %v", err)
			return nil, err
		}
	}

	return purchases, nil
}

// FindCompletedByPaymentOptionUUID finds all confirmed purchases made with a specific payment option UUID
func (self *PurchaseModel) FindCompletedByPaymentOptionUUID(ctx context.Context, paymentOptionUUID string) ([]*Purchase, error) {
	rows, err := self.db.Queries.FindCompletedPurchasesByPaymentOptionUUID(ctx, paymentOptionUUID)
	if err != nil {
		log.Printf("error finding completed purchases by payment option uuid %v: %v", paymentOptionUUID, err)
		return nil, err
	}

	purchases := make([]*Purchase, len(rows))
	for i, row := range rows {
		purchases[i], err = NewPurchase(self.db, self.models, &row)
		if err != nil {
			log.Printf("error creating purchase from row: %v", err)
			return nil, err
		}
	}

	return purchases, nil
}
