package models

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type WalletTrns struct {
	db          *db.Database
	models      *Models
	id          pgtype.UUID
	walletId    pgtype.UUID
	amount      float64
	newBalance  float64
	description string
	createdAt   time.Time
}

func NewWalletTrns(dtb *db.Database, mdls *Models) *WalletTrns {
	return &WalletTrns{
		db:     dtb,
		models: mdls,
	}
}

func (self *WalletTrns) Id() pgtype.UUID {
	return self.id
}

func (self *WalletTrns) WalletId() pgtype.UUID {
	return self.walletId
}

func (self *WalletTrns) Amount() float64 {
	return self.amount
}

func (self *WalletTrns) NewBalance() float64 {
	return self.newBalance
}

func (self *WalletTrns) Description() string {
	return self.description
}

func (self *WalletTrns) CreatedAt() time.Time {
	return self.createdAt
}

func (self *WalletTrns) UpdateTx(tx pgx.Tx, ctx context.Context, walletId pgtype.UUID, amount float64, newbal float64, desc string) error {
	err := self.db.Queries.UpdateWalletTrns(ctx, queries.UpdateWalletTrnsParams{
		WalletID:    walletId,
		Amount:      sdkutils.PgFloat64ToNumeric(amount),
		NewBalance:  sdkutils.PgFloat64ToNumeric(newbal),
		Description: desc,
		ID:          self.id,
	})
	if err != nil {
		log.Printf("error updating wallet transaction %+v: %v", self.id, err)
		return err
	}

	self.walletId = walletId
	self.amount = amount
	self.newBalance = newbal
	self.description = desc

	log.Printf("Succcessfully updated wallet transaction with id %v", walletId)
	return nil
}
