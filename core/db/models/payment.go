package models

import (
	"context"
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

func (self *Payment) ID() int64 {
	return self.id
}

func (self *Payment) PurchaseID() int64 {
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

func (self *Payment) Update(ctx context.Context, amt float64) error {
	err := self.db.Queries.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: amt,
		ID:     self.id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", self.id, err)
		return err
	}

	return nil
}
