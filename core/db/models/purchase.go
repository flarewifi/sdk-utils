package models

import (
	"context"
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
		purchase.uuid = p.Uuid
		purchase.deviceId = p.DeviceID
		purchase.sku = p.Sku
		purchase.name = p.Name
		purchase.description = p.Description
		purchase.price = p.Price
		purchase.anyPrice = p.AnyPrice
		purchase.callbackPluginPkg = p.CallbackPlugin
		purchase.callbackRoute = p.CallbackRoute
		purchase.webhookRoute = p.WebhookRoute
		purchase.processing = p.Processing
		purchase.paymentUrl = p.PaymentUrl

		metadata := make(map[string]string)
		if len(p.Metadata) > 0 {
			if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
				return nil, err
			}
		}

		purchase.metadata = metadata
		purchase.walletDebit = p.WalletDebit
		purchase.cancelledReason = &p.CancelledReason

		if p.WalletTxID.Valid {
			purchase.walletTxId = &p.WalletTxID.Int64
		}

		if p.ConfirmedAt.Valid {
			confirmedAt := p.ConfirmedAt.Time
			purchase.confirmedAt = &confirmedAt
		}
		if p.CancelledAt.Valid {
			cancelledAt := p.CancelledAt.Time
			purchase.cancelledAt = &cancelledAt
		}

		purchase.createdAt = p.CreatedAt
	}

	return purchase, nil
}

type Purchase struct {
	db                *db.Database
	models            *Models
	id                int64
	uuid              string
	deviceId          int64
	sku               string
	name              string
	description       string
	price             float64
	anyPrice          bool
	callbackPluginPkg string
	callbackRoute     string
	webhookRoute      string
	metadata          map[string]string
	walletDebit       float64
	walletTxId        *int64
	confirmedAt       *time.Time
	cancelledAt       *time.Time
	cancelledReason   *string
	processing        bool
	paymentUrl        string
	createdAt         time.Time
}

func (self *Purchase) ID() int64 {
	return self.id
}

func (self *Purchase) UUID() string {
	return self.uuid
}

func (self *Purchase) DeviceID() int64 {
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

func (self *Purchase) WalletTxID() *int64 {
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

func (self *Purchase) WebHookRoute() string {
	return self.webhookRoute
}

func (self *Purchase) Metadata() map[string]string {
	return self.metadata
}

func (self *Purchase) Processing() bool {
	return self.processing
}

func (self *Purchase) PaymentUrl() string {
	return self.paymentUrl
}

func (self *Purchase) IsConfirmed() bool {
	return self.confirmedAt != nil
}

func (self *Purchase) IsCancelled() bool {
	return self.cancelledAt != nil
}

func (self *Purchase) FixedPrice() (float64, bool) {
	return self.price, !self.anyPrice
}

func (self *Purchase) Device(ctx context.Context) (*Device, error) {
	return self.models.deviceModel.Find(ctx, self.deviceId)
}

func (self *Purchase) Confirm(ctx context.Context) error {
	dev, err := self.Device(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	wallet, err := dev.Wallet(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	var txid *int64
	dbt := self.walletDebit
	if dbt > 0 {
		newBal := wallet.Balance() - dbt
		err = wallet.Update(ctx, newBal)
		if err != nil {
			return errors.New("unable to update balance: " + err.Error())
		}

		desc := "Partial payment for " + self.description
		trns, err := self.models.walletTrnsModel.Create(ctx, CreateWalletTrnsParams{
			WalletID:    wallet.ID(),
			Amount:      -dbt,
			NewBalance:  newBal,
			Description: desc,
		})
		if err != nil {
			return err
		}

		id := trns.ID()
		txid = &id
	}

	now := time.Now().UTC()
	err = self.Update(ctx, dbt, txid, nil, &now, nil)
	if err != nil {
		return err
	}

	// Clear processing state when purchase is confirmed
	self.processing = false
	self.paymentUrl = ""

	return nil
}

func (self *Purchase) Cancel(ctx context.Context) error {
	dev, err := self.Device(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	pmtTotal, err := self.TotalPayment(ctx)
	if err != nil {
		return err
	}

	reason := "Cancelled purchase: " + self.description
	dbt := self.walletDebit
	cancelledAt := time.Now().UTC()

	if pmtTotal > 0 {
		wallet, err := dev.Wallet(ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		err = wallet.IncBalance(ctx, pmtTotal)
		if err != nil {
			log.Println("Error updating wallet balance: ", err)
			return err
		}

		trns, err := self.models.walletTrnsModel.Create(ctx, CreateWalletTrnsParams{
			WalletID:    wallet.ID(),
			Amount:      pmtTotal,
			NewBalance:  wallet.Balance(),
			Description: "Refund for " + reason,
		})
		if err != nil {
			log.Println(err)
			return err
		}

		trnsId := trns.ID()
		err = self.Update(ctx, dbt, &trnsId, &cancelledAt, nil, &reason)
		if err != nil {
			return err
		}
	} else {
		err = self.Update(ctx, dbt, nil, &cancelledAt, nil, &reason)
		if err != nil {
			return err
		}
	}

	// Clear processing state when purchase is cancelled
	self.processing = false
	self.paymentUrl = ""

	return nil
}

func (self *Purchase) Payments(ctx context.Context) ([]*Payment, error) {
	return self.models.paymentModel.FindAllByPurchase(ctx, self.id)
}

func (self *Purchase) TotalPayment(ctx context.Context) (float64, error) {
	pmts, err := self.Payments(ctx)
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

func (self *Purchase) Update(ctx context.Context, dbt float64, wtxID *int64, cancelledAt, confirmedAt *time.Time, reason *string) error {
	err := self.models.purchaseModel.Update(ctx, UpdatePurchaseParams{
		ID:              self.id,
		WalletDebit:     dbt,
		WalletTxID:      wtxID,
		CancelledAt:     cancelledAt,
		ConfirmedAt:     confirmedAt,
		CancelledReason: reason,
		Processing:      self.processing,
		PaymentUrl:      self.paymentUrl,
	})
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

func (self *Purchase) SetProcessing(ctx context.Context, paymentUrl string) error {
	// If paymentUrl is empty, clear processing state
	// If paymentUrl is provided, set processing to true
	processing := paymentUrl != ""

	err := self.models.purchaseModel.Update(ctx, UpdatePurchaseParams{
		ID:              self.id,
		WalletDebit:     self.walletDebit,
		WalletTxID:      self.walletTxId,
		CancelledAt:     self.cancelledAt,
		ConfirmedAt:     self.confirmedAt,
		CancelledReason: self.cancelledReason,
		Processing:      processing,
		PaymentUrl:      paymentUrl,
	})
	if err != nil {
		return err
	}

	self.processing = processing
	self.paymentUrl = paymentUrl

	return nil
}
