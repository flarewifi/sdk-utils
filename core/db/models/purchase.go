package models

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
		purchase.price = sdkutils.PgNumericToFloat64(p.Price)
		purchase.anyPrice = p.AnyPrice
		purchase.callbackPluginPkg = p.CallbackPlugin
		purchase.callbackRoute = p.CallbackRoute

		metadata := make(map[string]string)
		if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
			return nil, err
		}

		purchase.metadata = metadata
		purchase.walletDebit = sdkutils.PgNumericToFloat64(p.WalletDebit)
		purchase.cancelledReason = &p.CancelledReason

		if p.WalletTxID.Valid {
			purchase.walletTxId = &p.WalletTxID
		}

		if p.ConfirmedAt.Valid {
			purchase.confirmedAt = &p.ConfirmedAt.Time
		}
		if p.CancelledAt.Valid {
			purchase.cancelledAt = &p.CancelledAt.Time
		}
		if p.CreatedAt.Valid {
			purchase.createdAt = p.CreatedAt.Time
		}
	}

	return purchase, nil
}

type Purchase struct {
	db                *db.Database
	models            *Models
	id                pgtype.UUID
	deviceId          pgtype.UUID
	sku               string
	name              string
	description       string
	price             float64
	anyPrice          bool
	callbackPluginPkg string
	callbackRoute     string
	metadata          map[string]string
	walletDebit       float64
	walletTxId        *pgtype.UUID
	confirmedAt       *time.Time
	cancelledAt       *time.Time
	cancelledReason   *string
	createdAt         time.Time
}

func (self *Purchase) Id() pgtype.UUID {
	return self.id
}

func (self *Purchase) DeviceId() pgtype.UUID {
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

func (self *Purchase) WalletTxId() *pgtype.UUID {
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

func (self *Purchase) Device(tx pgx.Tx, ctx context.Context) (*Device, error) {
	dev, err := self.models.deviceModel.Find(ctx, self.deviceId)
	return dev, err
}

func (self *Purchase) Confirm(tx pgx.Tx, ctx context.Context) error {
	dev, err := self.Device(tx, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	wallet, err := dev.Wallet(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	var txid *pgtype.UUID
	dbt := self.walletDebit
	if dbt > 0 {
		newBal := wallet.Balance() - dbt
		err = wallet.UpdateTx(tx, ctx, newBal)
		if err != nil {
			return nil
		}

		desc := "Partial payment for " + self.description
		trns, err := self.models.walletTrnsModel.Create(ctx, wallet.Id(), -dbt, newBal, desc)
		if err != nil {
			return err
		}

		id := trns.Id()
		txid = &id
	}

	now := time.Now()
	return self.Update(tx, ctx, dbt, txid, nil, &now, nil)
}

func (self *Purchase) Cancel(tx pgx.Tx, ctx context.Context) error {
	dev, err := self.Device(tx, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	pmtTotal, err := self.TotalPaymentsTx(tx, ctx)
	if err != nil {
		return err
	}

	reason := "Cancelled purchase: " + self.description
	dbt := self.walletDebit
	cancelledAt := time.Now()

	if pmtTotal > 0 {
		wallet, err := dev.Wallet(ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		err = wallet.IncBalanceTx(tx, ctx, pmtTotal)
		if err != nil {
			log.Println("Error updating wallet balance: ", err)
			return err
		}

		trns, err := self.models.walletTrnsModel.Create(ctx, wallet.Id(), pmtTotal, wallet.Balance(), "Refund for "+reason)
		if err != nil {
			log.Println(err)
			return err
		}

		trnsId := trns.Id()
		return self.Update(tx, ctx, dbt, &trnsId, &cancelledAt, nil, &reason)
	}

	return self.Update(tx, ctx, dbt, nil, &cancelledAt, nil, &reason)
}

func (self *Purchase) PaymentsTx(tx pgx.Tx, ctx context.Context) ([]*Payment, error) {
	return self.models.paymentModel.FindAllByPurchase(ctx, self.id)
}

func (self *Purchase) TotalPaymentsTx(tx pgx.Tx, ctx context.Context) (float64, error) {
	pmts, err := self.PaymentsTx(tx, ctx)
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

func (self *Purchase) Update(tx pgx.Tx, ctx context.Context, dbt float64, trnsID *pgtype.UUID, cancelledAt, confirmedAt *time.Time, reason *string) error {
	err := self.models.purchaseModel.Update(tx, ctx, self.id, dbt, trnsID, cancelledAt, confirmedAt, reason)
	return err
}
