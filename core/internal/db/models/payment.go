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

type Payment struct {
	db         *db.Database
	models     *Models
	id         pgtype.UUID
	purchaseId pgtype.UUID
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

func (self *Payment) Id() pgtype.UUID {
	return self.id
}

func (self *Payment) PurchaseId() pgtype.UUID {
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
	err := self.db.Queries.UpdatePayment(ctx, sqlc.UpdatePaymentParams{
		Amount: pg.Float64ToNumeric(amt),
		ID:     self.id,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", self.id, err)
		return err
	}

	return nil
}
