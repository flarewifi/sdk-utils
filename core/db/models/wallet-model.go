package models

import (
	"context"
	"database/sql"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	"github.com/google/uuid"
)

type WalletModel struct {
	db        *db.Database
	models    *Models
	attrs     []string
	id        uuid.UUID
	balance   float64
	createdAt time.Time
}

func NewWalletModel(dtb *db.Database, mdls *Models) *WalletModel {
	attrs := []string{"id", "device_id", "balance", "created_at"}
	return &WalletModel{
		db:     dtb,
		models: mdls,
		attrs:  attrs,
	}
}

func (self *WalletModel) CreateTx(tx *sql.Tx, ctx context.Context, devId int64, bal float64) (*Wallet, error) {
	wId, err := self.db.Queries.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: devId,
		Balance:  bal,
	})
	if err != nil {
		log.Println("error creating wallet:", err)
		return nil, err
	}

	return self.Find(tx, ctx, wId)
}

func (self *WalletModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*Wallet, error) {
	qtx := self.db.Queries.WithTx(tx)
	w, err := qtx.FindWallet(ctx, id)
	if err != nil {
		log.Printf("error finding wallet %v: %v", id, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = w.Balance
	wallet.createdAt = w.CreatedAt

	return wallet, nil
}

func (self *WalletModel) Update(tx *sql.Tx, ctx context.Context, id int64, bal float64) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateWallet(ctx, queries.UpdateWalletParams{
		Balance: bal,
		ID:      id,
	})
	if err != nil {
		log.Printf("error updating wallet %v: %v\n", id, err)
		return err
	}

	self.balance = bal

	return nil
}

func (self *WalletModel) findByDevice(tx *sql.Tx, ctx context.Context, devId int64) (*Wallet, error) {
	qtx := self.db.Queries.WithTx(tx)
	w, err := qtx.FindWalletByDeviceId(ctx, devId)
	if err != nil {
		log.Printf("error finding wallet by device %v: %v", devId, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = w.Balance
	wallet.createdAt = w.CreatedAt

	return wallet, nil
}
