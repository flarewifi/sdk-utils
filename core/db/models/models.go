package models

import (
	"context"
	"core/db"
)

type Models struct {
	deviceModel            *DeviceModel
	deviceFingerprintModel *DeviceFingerprintModel
	sessionModel           *SessionModel
	purchaseModel          *PurchaseModel
	paymentModel           *PaymentModel
	walletModel            *WalletModel
	walletTrnsModel        *WalletTrnsModel
	logModel               *LogModel
	notificationModel      *NotificationModel
}

func New(dtb *db.Database) *Models {
	var models Models

	deviceModel := NewDeviceModel(dtb, &models)
	deviceFingerprintModel := NewDeviceFingerprintModel(dtb, &models)
	sessionModel := NewSessionModel(dtb, &models)
	purchaseModel := NewPurchaseModel(dtb, &models)
	paymentModel := NewPaymentModel(dtb, &models)
	walletModel := NewWalletModel(dtb, &models)
	walletTrnsModel := NewWalletTrnsModel(dtb, &models)
	logModel := NewLogModel(dtb, &models)

	models.deviceModel = deviceModel
	models.deviceFingerprintModel = deviceFingerprintModel
	models.sessionModel = sessionModel
	models.purchaseModel = purchaseModel
	models.paymentModel = paymentModel
	models.walletModel = walletModel
	models.walletTrnsModel = walletTrnsModel
	models.logModel = logModel
	models.notificationModel = NewNotificationModel(dtb, &models)

	return &models
}

func (self *Models) Device() *DeviceModel {
	return self.deviceModel
}

func (self *Models) DeviceFingerprint() *DeviceFingerprintModel {
	return self.deviceFingerprintModel
}

func (self *Models) Session() *SessionModel {
	return self.sessionModel
}

func (self *Models) Purchase() *PurchaseModel {
	return self.purchaseModel
}

func (self *Models) Payment() *PaymentModel {
	return self.paymentModel
}

func (self *Models) Wallet() *WalletModel {
	return self.walletModel
}

func (self *Models) WalletTrns() *WalletTrnsModel {
	return self.walletTrnsModel
}

func (self *Models) Log() *LogModel {
	return self.logModel
}

func (self *Models) Notification() *NotificationModel {
	return self.notificationModel
}

// Vacuum runs SQLite VACUUM to reclaim disk space after bulk delete operations.
// This should be called after operations that delete significant amounts of data.
func (self *Models) Vacuum(ctx context.Context) error {
	_, err := self.logModel.db.DB.ExecContext(ctx, "VACUUM")
	return err
}
