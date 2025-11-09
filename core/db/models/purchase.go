package models

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	"github.com/goccy/go-json"
)

func NewPurchase(dtb *db.Database, mdls *Models, p *queries.Purchase) (*Purchase, error) {
	purchase := &Purchase{
		db:     dtb,
		models: mdls,
	}

	if p != nil {
		purchase.id = p.ID
		purchase.deviceId = p.DeviceID
		purchase.sku = p.Sku
		purchase.name = p.Name
		purchase.description = p.Description
		purchase.price = p.Price
		purchase.anyPrice = p.AnyPrice
		purchase.callbackPluginPkg = p.CallbackPlugin
		purchase.callbackRoute = p.CallbackRoute

		metadata := make(map[string]string)
		if metadataBytes, ok := p.Metadata.([]byte); ok {
			if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
				return nil, err
			}
		}

		purchase.metadata = metadata
		purchase.walletDebit = p.WalletDebit
		purchase.cancelledReason = &p.CancelledReason

		if p.WalletTxID.Valid {
			purchase.walletTxId = &p.WalletTxID.Int64
		}

		if confirmedAt, ok := p.ConfirmedAt.(*time.Time); ok && confirmedAt != nil {
			purchase.confirmedAt = confirmedAt
		}
		if cancelledAt, ok := p.CancelledAt.(*time.Time); ok && cancelledAt != nil {
			purchase.cancelledAt = cancelledAt
		}

		purchase.createdAt = p.CreatedAt
	}

	return purchase, nil
}

type Purchase struct {
	db                *db.Database
	models            *Models
	id                int64
	deviceId          int64
	sku               string
	name              string
	description       string
	price             float64
	anyPrice          bool
	callbackPluginPkg string
	callbackRoute     string
	metadata          map[string]string
	walletDebit       float64
	walletTxId        *int64
	confirmedAt       *time.Time
	cancelledAt       *time.Time
	cancelledReason   *string
	createdAt         time.Time
}

func (self *Purchase) Id() int64 {
	return self.id
}

func (self *Purchase) DeviceId() int64 {
	return self.deviceId
}

func (self *Purchase) Sku() string {
	return self.sku
}

func (self *Purchase) Name() string {
	return self.name
}

func (self *Purchase) Description() string {
	return self.description
}

func (self *Purchase) Price() float64 {
	return self.price
}

func (self *Purchase) AnyPrice() bool {
	return self.anyPrice
}

func (self *Purchase) WalletDebit() float64 {
	return self.walletDebit
}

func (self *Purchase) WalletTxId() *int64 {
	return self.walletTxId
}

func (self *Purchase) ConfirmedAt() *time.Time {
	return self.confirmedAt
}

func (self *Purchase) CancelledAt() *time.Time {
	return self.cancelledAt
}

func (self *Purchase) CancelledReason() *string {
	return self.cancelledReason
}

func (self *Purchase) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Purchase) CallbackPluginPkg() string {
	return self.callbackPluginPkg
}

func (self *Purchase) CallbackRoute() string {
	return self.callbackRoute
}

func (self *Purchase) IsConfirmed() bool {
	return self.confirmedAt != nil
}

func (self *Purchase) IsCancelled() bool {
	return self.confirmedAt != nil
}

func (self *Purchase) FixedPrice() (float64, bool) {
	return self.price, !self.anyPrice
}

func (self *Purchase) Device(tx *sql.Tx, ctx context.Context) (*Device, error) {
	return self.models.deviceModel.Find(tx, ctx, self.deviceId)
}

func (self *Purchase) Confirm(tx *sql.Tx, ctx context.Context) error {
	dev, err := self.Device(tx, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	wallet, err := dev.Wallet(tx, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	var txid *int64
	dbt := self.walletDebit
	if dbt > 0 {
		newBal := wallet.Balance() - dbt
		err = wallet.Update(tx, ctx, newBal)
		if err != nil {
			return errors.New("unable to update balance: " + err.Error())
		}

		desc := "Partial payment for " + self.description
		trns, err := self.models.walletTrnsModel.Create(tx, ctx, wallet.Id(), -dbt, newBal, desc)
		if err != nil {
			return err
		}

		id := trns.Id()
		txid = &id
	}

	now := time.Now()
	return self.Update(tx, ctx, dbt, txid, nil, &now, nil)
}

func (self *Purchase) Cancel(tx *sql.Tx, ctx context.Context) error {
	dev, err := self.Device(tx, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	pmtTotal, err := self.TotalPayment(tx, ctx)
	if err != nil {
		return err
	}

	reason := "Cancelled purchase: " + self.description
	dbt := self.walletDebit
	cancelledAt := time.Now()

	if pmtTotal > 0 {
		wallet, err := dev.Wallet(tx, ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		err = wallet.IncBalance(tx, ctx, pmtTotal)
		if err != nil {
			log.Println("Error updating wallet balance: ", err)
			return err
		}

		trns, err := self.models.walletTrnsModel.Create(tx, ctx, wallet.Id(), pmtTotal, wallet.Balance(), "Refund for "+reason)
		if err != nil {
			log.Println(err)
			return err
		}

		trnsId := trns.Id()
		return self.Update(tx, ctx, dbt, &trnsId, &cancelledAt, nil, &reason)
	}

	return self.Update(tx, ctx, dbt, nil, &cancelledAt, nil, &reason)
}

func (self *Purchase) Payments(tx *sql.Tx, ctx context.Context) ([]*Payment, error) {
	return self.models.paymentModel.FindAllByPurchase(tx, ctx, self.id)
}

func (self *Purchase) TotalPayment(tx *sql.Tx, ctx context.Context) (float64, error) {
	pmts, err := self.Payments(tx, ctx)
	if err != nil {
		return 0, err
	}

	var total float64

	for _, p := range pmts {
		total += p.Amount()
	}

	total += self.WalletDebit()

	return total, nil
}

func (self *Purchase) Update(tx *sql.Tx, ctx context.Context, dbt float64, wtxID *int64, cancelledAt, confirmedAt *time.Time, reason *string) error {
	err := self.models.purchaseModel.Update(tx, ctx, self.id, dbt, wtxID, cancelledAt, confirmedAt, reason)
	if err != nil {
		return err
	}

	self.walletDebit = dbt
	self.walletTxId = wtxID
	self.cancelledAt = cancelledAt
	self.confirmedAt = confirmedAt
	self.cancelledReason = reason

	return nil
}
