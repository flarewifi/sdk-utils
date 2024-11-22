package models

import (
	"context"
	"log"
	"time"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (self *WalletModel) CreateTx(tx pgx.Tx, ctx context.Context, devId pgtype.UUID, bal float64) (*Wallet, error) {
	wId, err := self.db.Queries.CreateWallet(ctx, sqlc.CreateWalletParams{
		DeviceID: devId,
		Balance:  pg.Float64ToNumeric(bal),
	})
	if err != nil {
		log.Println("error creating wallet:", err)
		return nil, err
	}

	return self.Find(ctx, wId)
}

func (self *WalletModel) Find(ctx context.Context, id pgtype.UUID) (*Wallet, error) {
	w, err := self.db.Queries.FindWallet(ctx, id)
	if err != nil {
		log.Printf("error finding wallet %v: %v", id, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = pg.NumericToFloat64(w.Balance)
	wallet.createdAt = w.CreatedAt.Time

	return wallet, nil
}

func (self *WalletModel) Update(ctx context.Context, id pgtype.UUID, bal float64) error {
	err := self.db.Queries.UpdateWallet(ctx, sqlc.UpdateWalletParams{
		Balance: pg.Float64ToNumeric(bal),
		ID:      id,
	})
	if err != nil {
		log.Printf("error updating wallet %v: %v\n", id, err)
		return err
	}

	self.balance = bal

	return nil
}

func (self *WalletModel) findByDevice(ctx context.Context, devId pgtype.UUID) (*Wallet, error) {
	w, err := self.db.Queries.FindWalletByDeviceId(ctx, devId)
	if err != nil {
		log.Printf("error finding wallet by device %v: %v", devId, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = pg.NumericToFloat64(w.Balance)
	wallet.createdAt = w.CreatedAt.Time

	return wallet, nil
}
