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

	sdkutils "github.com/flarehotspot/sdk-utils"
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
	Metadata       map[string]string
}

// UpdatePurchaseParams holds parameters for updating a purchase
type UpdatePurchaseParams struct {
	ID              int64
	WalletDebit     float64
	WalletTxID      *int64
	CancelledAt     *time.Time
	ConfirmedAt     *time.Time
	CancelledReason *string
}

func NewPurchaseModel(dtb *db.Database, mdls *Models) *PurchaseModel {
	return &PurchaseModel{dtb, mdls}
}

func (self *PurchaseModel) Create(tx *sql.Tx, ctx context.Context, params CreatePurchaseParams) (*Purchase, error) {
	b, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, err
	}

	queryParams := queries.CreatePurchaseParams{
		DeviceID:       params.DeviceID,
		Sku:            params.SKU,
		Name:           params.Name,
		Description:    params.Description,
		Price:          params.Price,
		AnyPrice:       params.AnyPrice,
		CallbackPlugin: params.CallbackPlugin,
		CallbackRoute:  params.CallbackRoute,
		Metadata:       b,
	}

	fmt.Printf("Create Purchase: %+v\n", queryParams)

	qtx := self.db.Queries.WithTx(tx)
	pId, err := qtx.CreatePurchase(ctx, queryParams)
	if err != nil {
		log.Println("error creating purchase: %w", err)
		return nil, err
	}

	return self.Find(tx, ctx, pId)
}

func (self *PurchaseModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPurchase(ctx, id)
	if err != nil {
		log.Println("error finding purchase: %w", err)
		return nil, err
	}
	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) PendingPurchase(tx *sql.Tx, ctx context.Context, deviceId int64) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPendingPurchase(ctx, deviceId)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) FindByDeviceId(tx *sql.Tx, ctx context.Context, deviceId int64) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPurchaseByDeviceId(ctx, deviceId)
	if err != nil {
		log.Printf("error finding purchase by device id %v: %v", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) Update(tx *sql.Tx, ctx context.Context, params UpdatePurchaseParams) error {
	var cancellReason string
	if params.CancelledReason != nil {
		cancellReason = *params.CancelledReason
	}

	var walletTxID sql.NullInt64
	if params.WalletTxID != nil {
		walletTxID = sql.NullInt64{Int64: *params.WalletTxID, Valid: true}
	}

	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePurchase(ctx, queries.UpdatePurchaseParams{
		WalletDebit:     params.WalletDebit,
		WalletTxID:      walletTxID,
		CancelledAt:     sdkutils.TimeToNullTime(params.CancelledAt),
		ConfirmedAt:     sdkutils.TimeToNullTime(params.ConfirmedAt),
		CancelledReason: cancellReason,
		ID:              params.ID,
	})
	if err != nil {
		log.Printf("error updating purchase %v: %v", params.ID, err)
		return err
	}

	return nil
}
