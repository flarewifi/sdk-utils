package models

import (
	"context"
	"database/sql"
	"errors"
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

func (self *Wallet) ID() int64 {
	return self.id
}

func (self *Wallet) DeviceID() int64 {
	return self.deviceId
}

func (self *Wallet) Balance() float64 {
	return self.balance
}

func (self *Wallet) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Wallet) IncBalance(ctx context.Context, bal float64) error {
	newbal := self.balance + bal
	err := self.Update(ctx, newbal)
	if err != nil {
		return err
	}

	self.balance = newbal
	return nil
}

func (self *Wallet) Update(ctx context.Context, bal float64) error {
	err := self.models.walletModel.Update(ctx, UpdateWalletParams{
		ID:      self.id,
		Balance: bal,
	})
	if err != nil {
		return err
	}
	self.balance = bal
	return nil
}

func (self *Wallet) AvailableBal(ctx context.Context) (float64, error) {
	pending, err := self.models.purchaseModel.PendingPurchase(ctx, self.deviceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return self.balance, nil
		}
		return 0, err
	}

	dbt := pending.WalletDebit()
	return self.balance - dbt, nil
}
