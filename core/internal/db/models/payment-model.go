package models

import (
	"context"
	"log"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/jackc/pgx/v5/pgtype"
)

type PaymentModel struct {
	db     *db.Database
	models *Models
}

func NewPaymentModel(dtb *db.Database, mdls *Models) *PaymentModel {
	return &PaymentModel{dtb, mdls}
}

func (self *PaymentModel) Create(ctx context.Context, purid pgtype.UUID, amt float64, mtd string) (*Payment, error) {
	pId, err := self.db.Queries.CreatePayment(ctx, sqlc.CreatePaymentParams{
		PurchaseID: purid,
		Amount:     pg.Float64ToNumeric(amt),
		Optname:    mtd,
	})
	if err != nil {
		log.Println("error creating payment:", err)
		return nil, err
	}

	p, err := self.db.Queries.FindPayment(ctx, pId)
	if err != nil {
		log.Printf("error finding payemnt %v: %v", pId, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = pg.NumericToFloat64(p.Amount)
	payment.optname = p.Optname
	payment.createdAt = p.CreatedAt.Time

	return payment, nil
}

func (self *PaymentModel) Find(ctx context.Context, id pgtype.UUID) (*Payment, error) {
	p, err := self.db.Queries.FindPayment(ctx, id)
	if err != nil {
		log.Printf("error finding payment %v: %v", id, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = pg.NumericToFloat64(p.Amount)
	payment.optname = p.Optname
	payment.createdAt = p.CreatedAt.Time

	return payment, nil
}

func (self *PaymentModel) FindAllByPurchase(ctx context.Context, purId pgtype.UUID) ([]*Payment, error) {
	payments := []*Payment{}

	pRows, err := self.db.Queries.FindAllPaymentsByPurchaseId(ctx, purId)
	if err != nil {
		log.Printf("error finding payments by purchase id %v: %v", purId, err)
		return nil, err
	}

	// Parse payments
	for _, p := range pRows {
		nP := NewPayment(self.db, self.models)
		nP.id = p.ID
		nP.purchaseId = p.PurchaseID
		nP.amount = pg.NumericToFloat64(p.Amount)
		nP.optname = p.Optname
		nP.createdAt = p.CreatedAt.Time
		payments = append(payments, nP)
	}

	return payments, nil
}

func (self *PaymentModel) Update(ctx context.Context, id pgtype.UUID, amt float64, dbt *float64, txid *int64) error {
	err := self.db.Queries.UpdatePayment(ctx, sqlc.UpdatePaymentParams{
		Amount: pg.Float64ToNumeric(amt),
		ID:     id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", id, err)
		return err
	}

	return nil
}
