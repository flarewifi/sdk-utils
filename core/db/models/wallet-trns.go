package models

import (
	"context"
	"time"

	"core/db"
	"core/db/queries"
)

type WalletTrns struct {
	db          *db.Database
	models      *Models
	id          int64
	walletId    int64
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

func (self *WalletTrns) ID() int64 {
	return self.id
}

func (self *WalletTrns) WalletID() int64 {
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

func (self *WalletTrns) UpdateTx(ctx context.Context, walletId int64, amount float64, newbal float64, desc string) error {
	err := self.db.Queries.UpdateWalletTrns(ctx, queries.UpdateWalletTrnsParams{
		WalletID:    walletId,
		Amount:      amount,
		NewBalance:  newbal,
		Description: desc,
		ID:          self.id,
	})
	if err != nil {
		return err
	}

	self.walletId = walletId
	self.amount = amount
	self.newBalance = newbal
	self.description = desc

	return nil
}
