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

// CreateWalletParams holds parameters for creating a new wallet
type CreateWalletParams struct {
	DeviceID int64
	Balance  float64
}

// UpdateWalletParams holds parameters for updating a wallet
type UpdateWalletParams struct {
	ID      int64
	Balance float64
}

func NewWalletModel(dtb *db.Database, mdls *Models) *WalletModel {
	attrs := []string{"id", "device_id", "balance", "created_at"}
	return &WalletModel{
		db:     dtb,
		models: mdls,
		attrs:  attrs,
	}
}

func (self *WalletModel) CreateTx(tx *sql.Tx, ctx context.Context, params CreateWalletParams) (*Wallet, error) {
	wId, err := self.db.Queries.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: params.DeviceID,
		Balance:  params.Balance,
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

func (self *WalletModel) Update(tx *sql.Tx, ctx context.Context, params UpdateWalletParams) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateWallet(ctx, queries.UpdateWalletParams{
		Balance: params.Balance,
		ID:      params.ID,
	})
	if err != nil {
		log.Printf("error updating wallet %v: %v\n", params.ID, err)
		return err
	}

	self.balance = params.Balance

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
