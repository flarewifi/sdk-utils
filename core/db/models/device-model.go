package models

import (
	"context"
	"database/sql"
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
	MacAddress  string
	Ipv4Address string
	Ipv6Address string
	Hostname    string
}

// UpdateDeviceParams holds parameters for updating a device
type UpdateDeviceParams struct {
	ID          int64
	MacAddress  string
	Ipv4Address string
	Ipv6Address string
	Hostname    string
	UUID        string
	Status      int
}

func NewDeviceModel(database *db.Database, mdls *Models) *DeviceModel {
	return &DeviceModel{db: database, models: mdls}
}

// validateDeviceFields checks that required device fields are not blank
func validateDeviceFields(uuid, ipv4, ipv6, mac string) error {
	if strings.TrimSpace(uuid) == "" {
		return fmt.Errorf("uuid cannot be blank")
	}
	if strings.TrimSpace(ipv4) == "" && strings.TrimSpace(ipv6) == "" {
		return fmt.Errorf("at least one IP address (IPv4 or IPv6) is required")
	}
	if strings.TrimSpace(mac) == "" {
		return fmt.Errorf("mac address cannot be blank")
	}
	return nil
}

func (self *DeviceModel) Create(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	uid := sdkutils.NewUUID()

	// Validate required fields
	if err := validateDeviceFields(uid, params.Ipv4Address, params.Ipv6Address, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return nil, err
	}

	// CRITICAL: Check if this MAC is already marked as current for another device
	// This prevents creating duplicate devices for the same MAC address
	existingDeviceID, err := self.models.DeviceMac().FindDeviceByMac(ctx, params.MacAddress)
	if err == nil && existingDeviceID > 0 {
		return nil, fmt.Errorf("MAC address %s is already registered to device %d (cannot create duplicate device)", params.MacAddress, existingDeviceID)
	}

	dId, err := self.db.Queries.CreateDevice(ctx, queries.CreateDeviceParams{
		Ipv4Addr: params.Ipv4Address,
		Ipv6Addr: params.Ipv6Address,
		Hostname: params.Hostname,
		Uuid:     uid,
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
		macaddr:   params.MacAddress, // Store in memory for now
		ipv4addr:  d.Ipv4Addr,
		ipv6addr:  d.Ipv6Addr,
		hostname:  d.Hostname,
		createdAt: d.CreatedAt.Time,
		updatedAt: d.UpdatedAt.Time,
		status:    sdkapi.DeviceStatus(d.Status),
	}

	// Record initial MAC address in device_macs table
	_, err = self.models.DeviceMac().Create(ctx, queries.CreateDeviceMacParams{
		DeviceID:   dId,
		MacAddress: params.MacAddress,
		IsCurrent:  true,
	})
	if err != nil {
		// CRITICAL: If MAC recording fails, delete the device to maintain consistency
		log.Printf("[DeviceModel.Create] ERROR: Failed to record initial MAC address for device %d: %v", dId, err)
		log.Printf("[DeviceModel.Create] Rolling back device creation (deleting device %d)", dId)
		if delErr := self.db.Queries.DeleteDevice(ctx, dId); delErr != nil {
			log.Printf("[DeviceModel.Create] ERROR: Failed to rollback device %d after MAC record failure: %v", dId, delErr)
		}
		return nil, fmt.Errorf("failed to record MAC address for new device: %w", err)
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
	device.macaddr = d.MacAddress // Now comes from JOIN
	device.ipv4addr = d.Ipv4Addr
	device.ipv6addr = d.Ipv6Addr
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.updatedAt = d.UpdatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) FindByMac(ctx context.Context, mac string) (*Device, error) {
	// Find device by MAC using device_macs table
	deviceID, err := self.models.DeviceMac().FindDeviceByMac(ctx, mac)
	if err != nil {
		log.Printf("error finding device by MAC %s: %v", mac, err)
		return nil, err
	}

	// Now fetch the full device
	return self.Find(ctx, deviceID)
}

func (self *DeviceModel) FindByUUID(ctx context.Context, uid string) (*Device, error) {
	d, err := self.db.Queries.FindDeviceByUUID(ctx, uid)
	if err != nil {
		log.Printf("error finding device by UUID %s: %v", uid, err)
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress // Now comes from JOIN
	device.ipv4addr = d.Ipv4Addr
	device.ipv6addr = d.Ipv6Addr
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.updatedAt = d.UpdatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) FindByIp(ctx context.Context, ip string) (*Device, error) {
	d, err := self.db.Queries.FindDeviceByIp(ctx, ip)
	if err != nil {
		log.Printf("error finding device by IP %s: %v", ip, err)
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress // Now comes from JOIN
	device.ipv4addr = d.Ipv4Addr
	device.ipv6addr = d.Ipv6Addr
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.updatedAt = d.UpdatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) Update(ctx context.Context, params UpdateDeviceParams) error {
	// Validate required fields
	if err := validateDeviceFields(params.UUID, params.Ipv4Address, params.Ipv6Address, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return err
	}

	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		ID:       params.ID,
		Ipv4Addr: params.Ipv4Address,
		Ipv6Addr: params.Ipv6Address,
		Hostname: params.Hostname,
		Uuid:     params.UUID,
		Status:   int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", params.ID, err)
		return err
	}

	// Always record MAC address to update last_seen_at and ensure is_current is set
	// RecordMacAddress is idempotent - if MAC exists, it just updates timestamp
	log.Printf("[DeviceModel.Update] Recording MAC address %s for device %d", params.MacAddress, params.ID)
	err = self.models.DeviceMac().RecordMacAddress(ctx, params.ID, params.MacAddress)
	if err != nil {
		log.Printf("[DeviceModel.Update] ERROR: Failed to record MAC address: %v", err)
		// Don't fail the entire update, but log the error
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
// This transfers all sessions, purchases, fingerprints, MAC addresses, and wallet balance from source to target,
// then deletes the source device. Used when MAC randomization creates duplicate device records
// for the same physical device (validated by fingerprint matching).
func (self *DeviceModel) MergeDevices(ctx context.Context, targetDeviceID, sourceDeviceID int64) error {
	log.Printf("[DeviceModel.MergeDevices] Starting merge: source=%d -> target=%d", sourceDeviceID, targetDeviceID)

	// Start a transaction with IMMEDIATE mode (LevelSerializable maps to BEGIN
	// IMMEDIATE in the SQLite drivers).  MergeDevices is an exclusively write
	// operation, so acquiring the write lock upfront avoids the DEFERRED
	// read→write upgrade race that causes "database is locked" (SQLITE_BUSY).
	tx, err := self.db.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
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
		TargetDeviceID: sql.NullInt64{Int64: targetDeviceID, Valid: true},
		SourceDeviceID: sql.NullInt64{Int64: sourceDeviceID, Valid: true},
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

	// 4. Transfer MAC address records from source to target
	// First, delete any conflicting MAC records on target that also exist on source
	// This prevents unique constraint violations during transfer
	log.Printf("[DeviceModel.MergeDevices] Deleting conflicting MAC addresses...")
	err = qtx.DeleteConflictingMacsBeforeTransfer(ctx, queries.DeleteConflictingMacsBeforeTransferParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to delete conflicting MAC addresses: %v", err)
		return fmt.Errorf("failed to delete conflicting MAC addresses: %w", err)
	}

	log.Printf("[DeviceModel.MergeDevices] Transferring MAC addresses...")
	err = qtx.TransferMacs(ctx, queries.TransferMacsParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		log.Printf("[DeviceModel.MergeDevices] ERROR: Failed to transfer MAC addresses: %v", err)
		return fmt.Errorf("failed to transfer MAC addresses: %w", err)
	}

	// 5. Merge wallets - transfer balance and transactions
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

	// 6. Delete the source device
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

	// VACUUM is intentionally omitted here. The database operates in WAL mode,
	// which reclaims space automatically via periodic checkpointing. Calling
	// VACUUM in WAL mode forces a full database rewrite and blocks all concurrent
	// readers and writers for its duration — unacceptable in a merge job that may
	// process many pairs per run.

	return nil
}
