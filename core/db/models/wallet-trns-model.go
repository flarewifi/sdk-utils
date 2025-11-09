package models

import (
	"context"
	"database/sql"
	"log"

	"core/db"
	"core/db/queries"
)

type WalletTrnsModel struct {
	db     *db.Database
	models *Models
}

func NewWalletTrnsModel(dtb *db.Database, mdls *Models) *WalletTrnsModel {
	return &WalletTrnsModel{dtb, mdls}
}

func (self *WalletTrnsModel) Create(tx *sql.Tx, ctx context.Context, wltId int64, amount float64, newBal float64, desc string) (*WalletTrns, error) {
	qtx := self.db.Queries.WithTx(tx)
	wt, err := qtx.CreateWalletTrns(ctx, queries.CreateWalletTrnsParams{
		WalletID:    wltId,
		Amount:      amount,
		NewBalance:  newBal,
		Description: desc,
	})
	if err != nil {
		log.Println("error creating wallet transaction:", err)
		return nil, err
	}

	return &WalletTrns{
		db:          self.db,
		models:      self.models,
		id:          wt.ID,
		walletId:    wt.WalletID,
		amount:      amount,
		newBalance:  newBal,
		description: wt.Description,
		createdAt:   wt.CreatedAt,
	}, nil
}

func (self *WalletTrnsModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*WalletTrns, error) {
	qtx := self.db.Queries.WithTx(tx)
	wt, err := qtx.FindWalletTrns(ctx, id)
	if err != nil {
		log.Printf("error finding wallet transaction %v: %v\n", id, err)
		return nil, err
	}

	return &WalletTrns{
		db:          self.db,
		models:      self.models,
		id:          wt.ID,
		walletId:    wt.WalletID,
		amount:      wt.Amount,
		newBalance:  wt.NewBalance,
		description: wt.Description,
		createdAt:   wt.CreatedAt,
	}, nil
}
