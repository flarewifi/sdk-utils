package models

import (
	"context"
	"log"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/jackc/pgx/v5/pgtype"
)

type WalletTrnsModel struct {
	db     *db.Database
	models *Models
}

func NewWalletTrnsModel(dtb *db.Database, mdls *Models) *WalletTrnsModel {
	return &WalletTrnsModel{dtb, mdls}
}

func (self *WalletTrnsModel) Create(ctx context.Context, wltId pgtype.UUID, amount float64, newBal float64, desc string) (*WalletTrns, error) {
	wt, err := self.db.Queries.CreateWalletTrns(ctx, sqlc.CreateWalletTrnsParams{
		WalletID:    wltId,
		Amount:      pg.Float64ToNumeric(amount),
		NewBalance:  pg.Float64ToNumeric(newBal),
		Description: pgtype.Text{String: desc},
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
		amount:      pg.NumericToFloat64(wt.Amount),
		newBalance:  pg.NumericToFloat64(wt.NewBalance),
		description: wt.Description.String,
		createdAt:   wt.CreatedAt.Time,
	}, nil
}

func (self *WalletTrnsModel) Find(ctx context.Context, id pgtype.UUID) (*WalletTrns, error) {
	wt, err := self.db.Queries.FindWalletTrns(ctx, id)
	if err != nil {
		log.Printf("error finding wallet transaction %v: %v\n", id, err)
		return nil, err
	}

	return &WalletTrns{
		db:          self.db,
		models:      self.models,
		id:          wt.ID,
		walletId:    wt.WalletID,
		amount:      pg.NumericToFloat64(wt.Amount),
		newBalance:  pg.NumericToFloat64(wt.NewBalance),
		description: wt.Description.String,
		createdAt:   wt.CreatedAt.Time,
	}, nil
}
