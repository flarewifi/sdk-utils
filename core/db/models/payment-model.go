package models

import (
	"context"
	"log"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type PaymentModel struct {
	db     *db.Database
	models *Models
}

// CreatePaymentParams holds parameters for creating a new payment
type CreatePaymentParams struct {
	PurchaseID        int64
	Amount            float64
	PaymentOptionUUID string
	ProviderPkg       string
	ProviderName      string
}

// UpdatePaymentParams holds parameters for updating a payment
type UpdatePaymentParams struct {
	ID     int64
	Amount float64
}

func NewPaymentModel(dtb *db.Database, mdls *Models) *PaymentModel {
	return &PaymentModel{dtb, mdls}
}

func (self *PaymentModel) Create(ctx context.Context, params CreatePaymentParams) (*Payment, error) {
	// Generate UUID for the payment
	paymentUUID := sdkutils.NewUUID()

	pId, err := self.db.Queries.CreatePayment(ctx, queries.CreatePaymentParams{
		Uuid:              paymentUUID,
		PurchaseID:        params.PurchaseID,
		Amount:            params.Amount,
		PaymentOptionUuid: params.PaymentOptionUUID,
		ProviderPkg:       params.ProviderPkg,
		ProviderName:      params.ProviderName,
	})
	if err != nil {
		log.Println("error creating payment:", err)
		return nil, err
	}

	p, err := self.db.Queries.FindPayment(ctx, pId)
	if err != nil {
		log.Printf("error finding payemnt %v: %v", pId, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.uuid = p.Uuid
	payment.purchaseId = p.PurchaseID
	payment.amount = p.Amount
	payment.paymentOptionUUID = p.PaymentOptionUuid
	payment.providerPkg = p.ProviderPkg
	payment.providerName = p.ProviderName
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) Find(ctx context.Context, id int64) (*Payment, error) {
	p, err := self.db.Queries.FindPayment(ctx, id)
	if err != nil {
		log.Printf("error finding payment %v: %v", id, err)
		return nil, err
	}

	payment := NewPayment(self.db, self.models)
	payment.id = p.ID
	payment.uuid = p.Uuid
	payment.purchaseId = p.PurchaseID
	payment.amount = p.Amount
	payment.paymentOptionUUID = p.PaymentOptionUuid
	payment.providerPkg = p.ProviderPkg
	payment.providerName = p.ProviderName
	payment.createdAt = p.CreatedAt

	return payment, nil
}

func (self *PaymentModel) FindAllByPurchase(ctx context.Context, purId int64) ([]*Payment, error) {
	payments := []*Payment{}
	pRows, err := self.db.Queries.FindAllPaymentsByPurchaseId(ctx, purId)
	if err != nil {
		log.Printf("error finding payments by purchase id %v: %v", purId, err)
		return nil, err
	}

	// Parse payments
	for _, p := range pRows {
		nP := NewPayment(self.db, self.models)
		nP.id = p.ID
		nP.uuid = p.Uuid
		nP.purchaseId = p.PurchaseID
		nP.amount = p.Amount
		nP.paymentOptionUUID = p.PaymentOptionUuid
		nP.providerPkg = p.ProviderPkg
		nP.providerName = p.ProviderName
		nP.createdAt = p.CreatedAt
		payments = append(payments, nP)
	}

	return payments, nil
}

func (self *PaymentModel) Update(ctx context.Context, params UpdatePaymentParams) error {
	err := self.db.Queries.UpdatePayment(ctx, queries.UpdatePaymentParams{
		Amount: params.Amount,
		ID:     params.ID,
	})
	if err != nil {
		log.Printf("error updating payment %v: %v", params.ID, err)
		return err
	}

	return nil
}
