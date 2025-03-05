package models

import (
	"context"
	"log"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PaymentModel struct {
	db     *db.Database
	models *Models
}

func NewPaymentModel(dtb *db.Database, mdls *Models) *PaymentModel {
	return &PaymentModel{dtb, mdls}
}

func (self *PaymentModel) Create(tx pgx.Tx, ctx context.Context, purid pgtype.UUID, amt float64, mtd string) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	pId, err := qtx.CreatePayment(ctx, queries.CreatePaymentParams{
		PurchaseID:    purid,
		Amount:        sdkutils.PgFloat64ToNumeric(amt),
		PaymentMethod: mtd,
	})
	if err != nil {
		log.Println("error creating payment:", err)
		return nil, err
	}

	p, err := qtx.FindPayment(ctx, pId)
	if err != nil {
		log.Printf("error finding payemnt %v: %v", pId, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = sdkutils.PgNumericToFloat64(p.Amount)
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt.Time

	return payment, nil
}

func (self *PaymentModel) Find(tx pgx.Tx, ctx context.Context, id pgtype.UUID) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPayment(ctx, id)
	if err != nil {
		log.Printf("error finding payment %v: %v", id, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = sdkutils.PgNumericToFloat64(p.Amount)
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt.Time

	return payment, nil
}

func (self *PaymentModel) FindAllByPurchase(tx pgx.Tx, ctx context.Context, purId pgtype.UUID) ([]*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	payments := []*Payment{}
	pRows, err := qtx.FindAllPaymentsByPurchaseId(ctx, purId)
	if err != nil {
		log.Printf("error finding payments by purchase id %v: %v", purId, err)
		return nil, err
	}

	// Parse payments
	for _, p := range pRows {
		nP := NewPayment(self.db, self.models)
		nP.id = p.ID
		nP.purchaseId = p.PurchaseID
		nP.amount = sdkutils.PgNumericToFloat64(p.Amount)
		nP.optname = p.PaymentMethod
		nP.createdAt = p.CreatedAt.Time
		payments = append(payments, nP)
	}

	return payments, nil
}

func (self *PaymentModel) Update(tx pgx.Tx, ctx context.Context, id pgtype.UUID, amt float64, dbt *float64, txid *int64) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: sdkutils.PgFloat64ToNumeric(amt),
		ID:     id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", id, err)
		return err
	}

	return nil
}
