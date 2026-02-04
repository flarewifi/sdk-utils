package models

import (
	"context"
	"fmt"
	"log"
	"strings"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	sdkapi "sdk/api"
)

type DeviceModel struct {
	db     *db.Database
	models *Models
}

// CreateDeviceParams holds parameters for creating a new device
type CreateDeviceParams struct {
	MacAddress string
	IpAddress  string
	Hostname   string
}

// UpdateDeviceParams holds parameters for updating a device
type UpdateDeviceParams struct {
	ID         int64
	MacAddress string
	IpAddress  string
	Hostname   string
	UUID       string
	Status     int
}

func NewDeviceModel(database *db.Database, mdls *Models) *DeviceModel {
	return &DeviceModel{db: database, models: mdls}
}

// validateDeviceFields checks that required device fields are not blank
func validateDeviceFields(uuid, ip, mac string) error {
	if strings.TrimSpace(uuid) == "" {
		return fmt.Errorf("uuid cannot be blank")
	}
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("ip address cannot be blank")
	}
	if strings.TrimSpace(mac) == "" {
		return fmt.Errorf("mac address cannot be blank")
	}
	return nil
}

func (self *DeviceModel) Create(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	uid := sdkutils.NewUUID()

	// Validate required fields
	if err := validateDeviceFields(uid, params.IpAddress, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return nil, err
	}

	dId, err := self.db.Queries.CreateDevice(ctx, queries.CreateDeviceParams{
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
		Uuid:       uid,
	})
	if err != nil {
		log.Println("error creating new device:", err)
		return nil, err
	}

	d, err := self.db.Queries.FindDevice(ctx, dId)
	if err != nil {
		log.Printf("error finding device %v: %v\n", dId, err)
		return nil, err
	}

	dev := &Device{
		db:        self.db,
		models:    self.models,
		id:        d.ID,
		uuid:      d.Uuid,
		macaddr:   d.MacAddress,
		ipaddr:    d.IpAddress,
		hostname:  d.Hostname,
		createdAt: d.CreatedAt.Time,
		status:    sdkapi.DeviceStatus(d.Status),
	}

	_, err = self.db.Queries.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: dId,
		Balance:  0.0,
	})
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (self *DeviceModel) Find(ctx context.Context, id int64) (*Device, error) {
	d, err := self.db.Queries.FindDevice(ctx, id)
	if err != nil {
		log.Printf("error finding device %v: %v", id, err)
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	// log.Printf("Found device: %+v", device)
	return device, nil
}

func (self *DeviceModel) FindByMac(ctx context.Context, mac string) (*Device, error) {
	device := NewDevice(self.db, self.models)
	d, err := self.db.Queries.FindDeviceByMac(ctx, mac)
	if err != nil {
		log.Printf("error finding device %s: %v", mac, err)
		return nil, err
	}

	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) FindByUUID(ctx context.Context, uid string) (*Device, error) {
	device := NewDevice(self.db, self.models)
	d, err := self.db.Queries.FindDeviceByUUID(ctx, uid)
	if err != nil {
		log.Printf("error finding device by UUID %s: %v", uid, err)
		return nil, err
	}

	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) Update(ctx context.Context, params UpdateDeviceParams) error {
	// Validate required fields
	if err := validateDeviceFields(params.UUID, params.IpAddress, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return err
	}

	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		ID:         params.ID,
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
		Uuid:       params.UUID,
		Status:     int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", params.ID, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", params.ID)
	return nil
}

// BackfillEmptyUUIDs generates UUIDs for all devices that have empty UUID fields
func (self *DeviceModel) BackfillEmptyUUIDs(ctx context.Context) error {
	devices, err := self.db.Queries.FindDevicesWithEmptyUUID(ctx)
	if err != nil {
		log.Printf("error finding devices with empty UUID: %v", err)
		return err
	}

	for _, d := range devices {
		uid := sdkutils.NewUUID()
		err := self.db.Queries.UpdateDeviceUUID(ctx, queries.UpdateDeviceUUIDParams{
			ID:   d.ID,
			Uuid: uid,
		})
		if err != nil {
			log.Printf("error updating UUID for device %v: %v", d.ID, err)
			return err
		}
		log.Printf("Generated UUID %s for device %v", uid, d.ID)
	}

	if len(devices) > 0 {
		log.Printf("Backfilled UUIDs for %d devices", len(devices))
	}

	return nil
}

// MergeDevices merges sourceDeviceID into targetDeviceID.
// This transfers all sessions, purchases, fingerprints, and wallet balance from source to target,
// then deletes the source device. Used when MAC randomization creates duplicate device records
// for the same physical device (validated by fingerprint matching).
func (self *DeviceModel) MergeDevices(ctx context.Context, targetDeviceID, sourceDeviceID int64) error {
	log.Printf("[DeviceModel.MergeDevices] Starting merge: source=%d -> target=%d", sourceDeviceID, targetDeviceID)

	// Start a transaction
	tx, err := self.db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to begin transaction: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := self.db.Queries.WithTx(tx)

	// 1. Transfer sessions from source to target
	log.Printf("[DeviceModel.MergeDevices] Transferring sessions...")
	err = qtx.TransferSessionsToDevice(ctx, queries.TransferSessionsToDeviceParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to transfer sessions: %v", err)
		return fmt.Errorf("failed to transfer sessions: %w", err)
	}

	// 2. Transfer purchases from source to target
	log.Printf("[DeviceModel.MergeDevices] Transferring purchases...")
	err = qtx.TransferPurchasesToDevice(ctx, queries.TransferPurchasesToDeviceParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to transfer purchases: %v", err)
		return fmt.Errorf("failed to transfer purchases: %w", err)
	}

	// 3. Transfer fingerprints from source to target
	log.Printf("[DeviceModel.MergeDevices] Transferring fingerprints...")
	err = qtx.TransferFingerprintsToDevice(ctx, queries.TransferFingerprintsToDeviceParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to transfer fingerprints: %v", err)
		return fmt.Errorf("failed to transfer fingerprints: %w", err)
	}

	// 4. Merge wallets - transfer balance and transactions
	log.Printf("[DeviceModel.MergeDevices] Merging wallets...")
	sourceWallet, err := qtx.FindWalletByDeviceId(ctx, sourceDeviceID)
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] WARN: Source wallet not found (may not exist): %v", err)
		// Continue - source device might not have a wallet
	} else {
		targetWallet, err := qtx.FindWalletByDeviceId(ctx, targetDeviceID)
		if err != nil {
			log.Printf("[DeviceModel.MergeDevices] ERROR: Target wallet not found: %v", err)
			return fmt.Errorf("failed to find target wallet: %w", err)
		}

		// Transfer wallet transactions from source to target wallet
		err = qtx.TransferWalletTransactions(ctx, queries.TransferWalletTransactionsParams{
			TargetWalletID: targetWallet.ID,
			SourceWalletID: sourceWallet.ID,
		})
		if err != nil {
			log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to transfer wallet transactions: %v", err)
			return fmt.Errorf("failed to transfer wallet transactions: %w", err)
		}

		// Add source wallet balance to target wallet
		if sourceWallet.Balance > 0 {
			log.Printf("[DeviceModel.MergeDevices] Adding balance %.2f from source wallet to target", sourceWallet.Balance)
			err = qtx.AddToWalletBalance(ctx, queries.AddToWalletBalanceParams{
				Amount:   sourceWallet.Balance,
				DeviceID: targetDeviceID,
			})
			if err != nil {
				log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to add wallet balance: %v", err)
				return fmt.Errorf("failed to add wallet balance: %w", err)
			}
		}

		// Delete source wallet (transactions already transferred)
		err = qtx.DeleteWalletByDeviceId(ctx, sourceDeviceID)
		if err != nil {
			log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to delete source wallet: %v", err)
			return fmt.Errorf("failed to delete source wallet: %w", err)
		}
	}

	// 5. Delete the source device
	log.Printf("[DeviceModel.MergeDevices] Deleting source device %d...", sourceDeviceID)
	err = qtx.DeleteDevice(ctx, sourceDeviceID)
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to delete source device: %v", err)
		return fmt.Errorf("failed to delete source device: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[DeviceModel.MergeDevices] SUCCESS: Merged device %d into device %d", sourceDeviceID, targetDeviceID)
	return nil
}
