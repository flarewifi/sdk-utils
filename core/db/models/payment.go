package models

import (
	"context"
	"database/sql"
	"log"
	"time"

	"core/db"
	"core/db/queries"
)

type Payment struct {
	db         *db.Database
	models     *Models
	id         int64
	purchaseId int64
	amount     float64
	optname    string
	createdAt  time.Time
}

func NewPayment(dtb *db.Database, mdls *Models) *Payment {
	return &Payment{
		db:     dtb,
		models: mdls,
	}
}

func (self *Payment) Id() int64 {
	return self.id
}

func (self *Payment) PurchaseId() int64 {
	return self.purchaseId
}

func (self *Payment) Amount() float64 {
	return self.amount
}

func (self *Payment) OptName() string {
	return self.optname
}

func (self *Payment) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Payment) Update(tx *sql.Tx, ctx context.Context, amt float64) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: amt,
		ID:     self.id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", self.id, err)
		return err
	}

	return nil
}
