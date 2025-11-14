package models

import (
	"core/db"
)

type Models struct {
	deviceModel         *DeviceModel
	sessionModel        *SessionModel
	purchaseModel       *PurchaseModel
	paymentModel        *PaymentModel
	walletModel         *WalletModel
	walletTrnsModel     *WalletTrnsModel
	logModel            *LogModel
	notificationModel   *NotificationModel
	quickAccessNavModel *QuickAccessNavModel
}

func New(dtb *db.Database) *Models {
	var models Models

	deviceModel := NewDeviceModel(dtb, &models)
	sessionModel := NewSessionModel(dtb, &models)
	purchaseModel := NewPurchaseModel(dtb, &models)
	paymentModel := NewPaymentModel(dtb, &models)
	walletModel := NewWalletModel(dtb, &models)
	walletTrnsModel := NewWalletTrnsModel(dtb, &models)
	logModel := NewLogModel(dtb, &models)

	models.deviceModel = deviceModel
	models.sessionModel = sessionModel
	models.purchaseModel = purchaseModel
	models.paymentModel = paymentModel
	models.walletModel = walletModel
	models.walletTrnsModel = walletTrnsModel
	models.logModel = logModel
	models.notificationModel = NewNotificationModel(dtb, &models)
	models.quickAccessNavModel = NewQuickAccessNavModel(dtb, &models)

	return &models
}

func (self *Models) Device() *DeviceModel {
	return self.deviceModel
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

func (self *Models) QuickAccessNav() *QuickAccessNavModel {
	return self.quickAccessNavModel
}
