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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PurchaseModel struct {
	db     *db.Database
	models *Models
}

func NewPurchaseModel(dtb *db.Database, mdls *Models) *PurchaseModel {
	return &PurchaseModel{dtb, mdls}
}

func (self *PurchaseModel) Create(tx pgx.Tx, ctx context.Context, deviceId pgtype.UUID, sku string, name string, desc string, price float64, vprice bool, pkg string, routename string, metadata map[string]string) (*Purchase, error) {
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	params := queries.CreatePurchaseParams{
		DeviceID:       deviceId,
		Sku:            sku,
		Name:           name,
		Description:    desc,
		Price:          sdkutils.PgFloat64ToNumeric(price),
		AnyPrice:       vprice,
		CallbackPlugin: pkg,
		CallbackRoute:  routename,
		Metadata:       b,
	}

	fmt.Printf("Create Purchase: %+v\n", params)

	qtx := self.db.Queries.WithTx(tx)
	pId, err := qtx.CreatePurchase(ctx, params)
	if err != nil {
		log.Println("error creating purchase: %w", err)
		return nil, err
	}

	return self.Find(tx, ctx, pId)
}

func (self *PurchaseModel) Find(tx pgx.Tx, ctx context.Context, id pgtype.UUID) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPurchase(ctx, id)
	if err != nil {
		log.Println("error finding purchase: %w", err)
		return nil, err
	}
	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) PendingPurchase(tx pgx.Tx, ctx context.Context, deviceId pgtype.UUID) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPendingPurchase(ctx, deviceId)
	if err != nil {
		log.Printf("error finding pending purchase with dev id %v: %v\n", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) FindByDeviceId(tx pgx.Tx, ctx context.Context, deviceId pgtype.UUID) (*Purchase, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPurchaseByDeviceId(ctx, deviceId)
	if err != nil {
		log.Printf("error finding purchase by device id %v: %v", deviceId, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}

func (self *PurchaseModel) Update(tx pgx.Tx, ctx context.Context, id pgtype.UUID, dbt float64, txid *pgtype.UUID, cancelledAt *time.Time, confirmedAt *time.Time, reason *string) error {
	var cancellReason string
	if reason != nil {
		cancellReason = *reason
	}

	var wtxid pgtype.UUID
	if txid != nil {
		wtxid = *txid
	}

	var cancelledAtTime, confirmedAtTime pgtype.Timestamp
	if cancelledAt != nil {
		cancelledAtTime = pgtype.Timestamp{Time: *cancelledAt, Valid: true}
	}
	if confirmedAt != nil {
		confirmedAtTime = pgtype.Timestamp{Time: *confirmedAt, Valid: true}
	}

	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePurchase(ctx, queries.UpdatePurchaseParams{
		WalletDebit:     sdkutils.PgFloat64ToNumeric(dbt),
		WalletTxID:      wtxid,
		CancelledAt:     cancelledAtTime,
		ConfirmedAt:     confirmedAtTime,
		CancelledReason: cancellReason,
		ID:              id,
	})
	if err != nil {
		log.Printf("error updating purchase %v: %v", id, err)
		return err
	}

	return nil
}
