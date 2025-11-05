package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type PaymentModel struct {
	db     *db.Database
	models *Models
}

func NewPaymentModel(dtb *db.Database, mdls *Models) *PaymentModel {
	return &PaymentModel{dtb, mdls}
}

func (self *PaymentModel) Create(tx *sql.Tx, ctx context.Context, purid int32, amt float64, mtd string) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	pId, err := qtx.CreatePayment(ctx, queries.CreatePaymentParams{
		PurchaseID:    purid,
		Amount:        fmt.Sprintf("%.6f", amt),
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

	amount, err := strconv.ParseFloat(p.Amount, 64)
	if err != nil {
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = amount
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) Find(tx *sql.Tx, ctx context.Context, id int32) (*Payment, error) {
	qtx := self.db.Queries.WithTx(tx)
	p, err := qtx.FindPayment(ctx, id)
	if err != nil {
		log.Printf("error finding payment %v: %v", id, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.purchaseId = p.PurchaseID
	payment.amount = sdkutils.StrToFloat64(p.Amount)
	payment.optname = p.PaymentMethod
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) FindAllByPurchase(tx *sql.Tx, ctx context.Context, purId int32) ([]*Payment, error) {
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
		nP.amount = sdkutils.StrToFloat64(p.Amount)
		nP.optname = p.PaymentMethod
		nP.createdAt = p.CreatedAt
		payments = append(payments, nP)
	}

	return payments, nil
}

func (self *PaymentModel) Update(tx *sql.Tx, ctx context.Context, id int32, amt float64, dbt *float64, txid *int64) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: sdkutils.Float64ToStr(amt),
		ID:     id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", id, err)
		return err
	}

	return nil
}
