package models

import (
	"context"
	"database/sql"
	"time"

	"core/db"
)

type Wallet struct {
	db        *db.Database
	models    *Models
	id        int64
	deviceId  int64
	balance   float64
	createdAt time.Time
}

func NewWallet(dtb *db.Database, m *Models) *Wallet {
	return &Wallet{
		db:     dtb,
		models: m,
	}
}

func (self *Wallet) Id() int64 {
	return self.id
}

func (self *Wallet) DeviceId() int64 {
	return self.deviceId
}

func (self *Wallet) Balance() float64 {
	return self.balance
}

func (self *Wallet) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Wallet) IncBalance(tx *sql.Tx, ctx context.Context, bal float64) error {
	newbal := self.balance + bal
	err := self.Update(tx, ctx, newbal)
	if err != nil {
		return err
	}

	self.balance = newbal
	return nil
}

func (self *Wallet) Update(tx *sql.Tx, ctx context.Context, bal float64) error {
	err := self.models.walletModel.Update(tx, ctx, self.id, bal)
	if err != nil {
		return err
	}
	self.balance = bal
	return nil
}

func (self *Wallet) AvailableBal(tx *sql.Tx, ctx context.Context) (float64, error) {
	pending, err := self.models.purchaseModel.PendingPurchase(tx, ctx, self.deviceId)
	if err != nil {
		return 0, nil
	}

	dbt := pending.WalletDebit()
	return self.balance - dbt, nil
}
