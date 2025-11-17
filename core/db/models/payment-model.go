package models

import (
	"context"
	"database/sql"
	"log"

	"core/db"
	"core/db/queries"
)

type PaymentModel struct {
	db     *db.Database
	models *Models
}

// CreatePaymentParams holds parameters for creating a new payment
type CreatePaymentParams struct {
	PurchaseID    int64
	Amount        float64
	PaymentMethod string
}

// UpdatePaymentParams holds parameters for updating a payment
type UpdatePaymentParams struct {
	ID     int64
	Amount float64
}

func NewPaymentModel(dtb *db.Database, mdls *Models) *PaymentModel {
	return &PaymentModel{dtb, mdls}
}

func (self *PaymentModel) Create(tx *sql.Tx, ctx context.Context, params CreatePaymentParams) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	pId, err := qtx.CreatePayment(ctx, queries.CreatePaymentParams{
		PurchaseID:    params.PurchaseID,
		Amount:        params.Amount,
		PaymentMethod: params.PaymentMethod,
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
	payment.amount = p.Amount
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPayment(ctx, id)
	if err != nil {
		log.Printf("error finding payment %v: %v", id, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = p.Amount
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) FindAllByPurchase(tx *sql.Tx, ctx context.Context, purId int64) ([]*Payment, error) {
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
		nP.amount = p.Amount
		nP.optname = p.PaymentMethod
		nP.createdAt = p.CreatedAt
		payments = append(payments, nP)
	}

	return payments, nil
}

func (self *PaymentModel) Update(tx *sql.Tx, ctx context.Context, params UpdatePaymentParams) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: params.Amount,
		ID:     params.ID,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", params.ID, err)
		return err
	}

	return nil
}
