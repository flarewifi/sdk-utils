package models

import (
	"context"
	"log"
	"time"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/jackc/pgx/v5/pgtype"
)

type PurchaseModel struct {
	db     *db.Database
	models *Models
	attrs  []string
}

func NewPurchaseModel(dtb *db.Database, mdls *Models) *PurchaseModel {
	attrs := []string{"id", "device_id", "sku", "name", "description", "price", "any_price", "callback_plugin", "callback_vue_route_name", "wallet_debit", "wallet_tx_id", "confirmed_at", "cancelled_at", "cancelled_reason", "created_at"}
	return &PurchaseModel{dtb, mdls, attrs}
}

func (self *PurchaseModel) Create(ctx context.Context, deviceId pgtype.UUID, sku string, name string, desc string, price float64, vprice bool, pkg string, routename string) (*Purchase, error) {
	pId, err := self.db.Queries.CreatePurchase(ctx, sqlc.CreatePurchaseParams{
		DeviceID:             deviceId,
		Sku:                  sku,
		Name:                 name,
		Description:          pgtype.Text{String: desc, Valid: desc != ""},
		Price:                pg.Float64ToNumeric(price),
		AnyPrice:             vprice,
		CallbackPlugin:       pkg,
		CallbackVueRouteName: pgtype.Text{String: routename, Valid: routename != ""},
	})
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

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = pg.NumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackVueRouteName = p.CallbackVueRouteName.String
	purchase.walletDebit = pg.NumericToFloat64(p.WalletDebit)
	purchase.walletTxId = &p.WalletTxID
	purchase.confirmedAt = &p.ConfirmedAt.Time
	purchase.cancelledAt = &p.CancelledAt.Time
	purchase.cancelledReason = &p.CancelledReason.String
	purchase.createdAt = p.CreatedAt.Time

	return purchase, err
}

func (self *PurchaseModel) PendingPurchase(ctx context.Context, deviceId pgtype.UUID) (*Purchase, error) {
	p, err := self.db.Queries.FindPending(ctx, deviceId)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = pg.NumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackVueRouteName = p.CallbackVueRouteName.String
	purchase.walletDebit = pg.NumericToFloat64(p.WalletDebit)
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

	purchase := NewPurchase(self.db, self.models)
	purchase.id = p.ID
	purchase.deviceId = p.DeviceID
	purchase.sku = p.Sku
	purchase.name = p.Name
	purchase.description = p.Description.String
	purchase.price = pg.NumericToFloat64(p.Price)
	purchase.anyPrice = p.AnyPrice
	purchase.callbackPluginPkg = p.CallbackPlugin
	purchase.callbackVueRouteName = p.CallbackVueRouteName.String
	purchase.walletDebit = pg.NumericToFloat64(p.WalletDebit)
	purchase.walletTxId = &p.WalletTxID
	purchase.confirmedAt = &p.ConfirmedAt.Time
	purchase.cancelledAt = &p.CancelledAt.Time
	purchase.cancelledReason = &p.CancelledReason.String
	purchase.createdAt = p.CreatedAt.Time

	return purchase, err
}

func (self *PurchaseModel) Update(ctx context.Context, id pgtype.UUID, dbt float64, txid *pgtype.UUID, cancelledAt *time.Time, confirmedAt *time.Time, reason *string) error {
	err := self.db.Queries.UpdatePurchase(ctx, sqlc.UpdatePurchaseParams{
		WalletDebit:     pg.Float64ToNumeric(dbt),
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
