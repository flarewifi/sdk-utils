package models

import (
	"context"
	"log"

	"core/db"
	"core/db/queries"
)

type WalletTrnsModel struct {
	db     *db.Database
	models *Models
}

// CreateWalletTrnsParams holds parameters for creating a new wallet transaction
type CreateWalletTrnsParams struct {
	WalletID    int64
	Amount      float64
	NewBalance  float64
	Description string
}

func NewWalletTrnsModel(dtb *db.Database, mdls *Models) *WalletTrnsModel {
	return &WalletTrnsModel{dtb, mdls}
}

func (self *WalletTrnsModel) Create(ctx context.Context, params CreateWalletTrnsParams) (*WalletTrns, error) {
	wt, err := self.db.Queries.CreateWalletTrns(ctx, queries.CreateWalletTrnsParams{
		WalletID:    params.WalletID,
		Amount:      params.Amount,
		NewBalance:  params.NewBalance,
		Description: params.Description,
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
		amount:      params.Amount,
		newBalance:  params.NewBalance,
		description: wt.Description,
		createdAt:   wt.CreatedAt,
	}, nil
}

func (self *WalletTrnsModel) Find(ctx context.Context, id int64) (*WalletTrns, error) {
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
		amount:      wt.Amount,
		newBalance:  wt.NewBalance,
		description: wt.Description,
		createdAt:   wt.CreatedAt,
	}, nil
}
