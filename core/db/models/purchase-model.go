package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgtype"
)

type PurchaseModel struct {
	db     *db.Database
	models *Models
}

func NewPurchaseModel(dtb *db.Database, mdls *Models) *PurchaseModel {
	return &PurchaseModel{dtb, mdls}
}

func (self *PurchaseModel) Create(ctx context.Context, deviceId pgtype.UUID, sku string, name string, desc string, price float64, vprice bool, pkg string, routename string, metadata map[string]string) (*Purchase, error) {

	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	params := queries.CreatePurchaseParams{
		DeviceID:       deviceId,
		Sku:            sku,
		Name:           name,
		Description:    pgtype.Text{String: desc, Valid: desc != ""},
		Price:          sdkutils.PgFloat64ToNumeric(price),
		AnyPrice:       vprice,
		CallbackPlugin: pkg,
		CallbackRoute:  routename,
		Metadata:       b,
	}

	fmt.Printf("Create Purchase: %+v\n", params)

	pId, err := self.db.Queries.CreatePurchase(ctx, params)
	if err != nil {
		log.Println("error creating purchase: %w", err)
		return nil, err
	}

	return self.Find(ctx, pId)
}

func (self *PurchaseModel) Find(ctx context.Context, id pgtype.UUID) (*Purchase, error) {
	p, err := self.db.Queries.FindPurchase(ctx, id)
	if err != nil {
		log.Println("error finding purchase: %w", err)
		return nil, err
	}

	metadata := make(map[string]string)
	if err = json.Unmarshal(p.Metadata, &metadata); err != nil {
		return nil, err
	}

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = sdkutils.PgNumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackRoute = p.CallbackRoute
	purchase.metadata = metadata
	purchase.walletDebit = sdkutils.PgNumericToFloat64(p.WalletDebit)
	purchase.walletTxId = &p.WalletTxID
	purchase.confirmedAt = &p.ConfirmedAt.Time
	purchase.cancelledAt = &p.CancelledAt.Time
	purchase.cancelledReason = &p.CancelledReason.String
	purchase.createdAt = p.CreatedAt.Time

	return purchase, err
}

func (self *PurchaseModel) PendingPurchase(ctx context.Context, deviceId pgtype.UUID) (*Purchase, error) {
	p, err := self.db.Queries.FindPendingPurchase(ctx, deviceId)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	metadata := make(map[string]string)
	if err = json.Unmarshal(p.Metadata, &metadata); err != nil {
		return nil, err
	}

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = sdkutils.PgNumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackRoute = p.CallbackRoute
	purchase.metadata = metadata
	purchase.walletDebit = sdkutils.PgNumericToFloat64(p.WalletDebit)
	purchase.walletTxId = &p.WalletTxID
	purchase.confirmedAt = &p.ConfirmedAt.Time
	purchase.cancelledAt = &p.CancelledAt.Time
	purchase.cancelledReason = &p.CancelledReason.String
	purchase.createdAt = p.CreatedAt.Time

	return purchase, err
}

func (self *PurchaseModel) FindByDeviceId(ctx context.Context, deviceId pgtype.UUID) (*Purchase, error) {
	p, err := self.db.Queries.FindPurchaseByDeviceId(ctx, deviceId)
	if err != nil {
		log.Printf("error finding purchase by device id %v: %v", deviceId, err)
		return nil, err
	}

	metadata := make(map[string]string)
	if err = json.Unmarshal(p.Metadata, &metadata); err != nil {
		return nil, err
	}

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = sdkutils.PgNumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackRoute = p.CallbackRoute
	purchase.metadata = metadata
	purchase.walletDebit = sdkutils.PgNumericToFloat64(p.WalletDebit)
	purchase.walletTxId = &p.WalletTxID
	purchase.confirmedAt = &p.ConfirmedAt.Time
	purchase.cancelledAt = &p.CancelledAt.Time
	purchase.cancelledReason = &p.CancelledReason.String
	purchase.createdAt = p.CreatedAt.Time

	return purchase, err
}

func (self *PurchaseModel) Update(ctx context.Context, id pgtype.UUID, dbt float64, txid *pgtype.UUID, cancelledAt *time.Time, confirmedAt *time.Time, reason *string) error {
	err := self.db.Queries.UpdatePurchase(ctx, queries.UpdatePurchaseParams{
		WalletDebit:     sdkutils.PgFloat64ToNumeric(dbt),
		WalletTxID:      *txid,
		CancelledAt:     pgtype.Timestamp{Time: *cancelledAt},
		ConfirmedAt:     pgtype.Timestamp{Time: *confirmedAt},
		CancelledReason: pgtype.Text{String: *reason},
		ID:              id,
	})
	if err != nil {
		log.Printf("error updating purchase %v: %v", id, err)
		return err
	}

	return nil
}
