package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		return nil, err
	}

	// CRITICAL: Check if this MAC is already marked as current for another device
	// This prevents creating duplicate devices for the same MAC address
	existingDeviceID, err := self.models.DeviceMac().FindDeviceByMac(ctx, params.MacAddress)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing device by MAC %s: %w", params.MacAddress, err)
	}
	if err == nil && existingDeviceID > 0 {
		return nil, fmt.Errorf("MAC address %s is already registered to device %d (cannot create duplicate device)", params.MacAddress, existingDeviceID)
	}

	cookieToken := sdkutils.NewUUID()

	// Wrap device + MAC + wallet creation in a single transaction.
	// If any step fails, the entire operation is rolled back automatically.
	var dev *Device
	txErr := sdkutils.RunInTx(self.db.DB, ctx, func(tx *sql.Tx) error {
		q := queries.New(tx)

		dId, err := q.CreateDevice(ctx, queries.CreateDeviceParams{
			Ipv4Addr:    params.Ipv4Address,
			Ipv6Addr:    params.Ipv6Address,
			Hostname:    params.Hostname,
			Uuid:        uid,
			CookieToken: cookieToken,
		})
		if err != nil {
			return err
		}

		d, err := q.FindDevice(ctx, dId)
		if err != nil {
			return err
		}

		dev = &Device{
			db:          self.db,
			models:      self.models,
			id:          d.ID,
			uuid:        d.Uuid,
			cookieToken: d.CookieToken,
			macaddr:     params.MacAddress,
			ipv4addr:    d.Ipv4Addr,
			ipv6addr:    d.Ipv6Addr,
			hostname:    d.Hostname,
			createdAt:   d.CreatedAt.Time,
			updatedAt:   d.UpdatedAt.Time,
			status:      sdkapi.DeviceStatus(d.Status),
		}

		_, err = q.CreateDeviceMac(ctx, queries.CreateDeviceMacParams{
			DeviceID:   dId,
			MacAddress: params.MacAddress,
			IsCurrent:  true,
		})
		if err != nil {
			return fmt.Errorf("failed to record MAC address for new device: %w", err)
		}

		_, err = q.CreateWallet(ctx, queries.CreateWalletParams{
			DeviceID: dId,
			Balance:  0.0,
		})
		if err != nil {
			return fmt.Errorf("failed to create wallet for new device: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return dev, nil
}

func (self *DeviceModel) Find(ctx context.Context, id int64) (*Device, error) {
	d, err := self.db.Queries.FindDevice(ctx, id)
	if err != nil {
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.cookieToken = d.CookieToken
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
		return nil, err
	}

	// Now fetch the full device
	return self.Find(ctx, deviceID)
}

func (self *DeviceModel) FindByUUID(ctx context.Context, uid string) (*Device, error) {
	d, err := self.db.Queries.FindDeviceByUUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.cookieToken = d.CookieToken
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
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.cookieToken = d.CookieToken
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
		return err
	}

	// Always record MAC address to update last_seen_at and ensure is_current is set
	// RecordMacAddress is idempotent - if MAC exists, it just updates timestamp
	if err := self.models.DeviceMac().RecordMacAddress(ctx, params.ID, params.MacAddress); err != nil {
		return fmt.Errorf("failed to record MAC address for device %d: %w", params.ID, err)
	}

	return nil
}

// BackfillEmptyUUIDs generates UUIDs for all devices that have empty UUID fields
func (self *DeviceModel) BackfillEmptyUUIDs(ctx context.Context) error {
	devices, err := self.db.Queries.FindDevicesWithEmptyUUID(ctx)
	if err != nil {
		return err
	}

	for _, d := range devices {
		uid := sdkutils.NewUUID()
		err := self.db.Queries.UpdateDeviceUUID(ctx, queries.UpdateDeviceUUIDParams{
			ID:   d.ID,
			Uuid: uid,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// MergeDevices merges sourceDeviceID into targetDeviceID.
// Transfers all data (sessions, purchases, fingerprints, MACs, wallet) from source
// to target, then deletes the source device.
func (self *DeviceModel) MergeDevices(ctx context.Context, targetDeviceID, sourceDeviceID int64) error {
	tx, err := self.db.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := self.db.Queries.WithTx(tx)

	// 1. Transfer sessions from source to target
	err = qtx.TransferSessionsToDevice(ctx, queries.TransferSessionsToDeviceParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		return fmt.Errorf("failed to transfer sessions: %w", err)
	}
	err = qtx.TransferPurchasesToDevice(ctx, queries.TransferPurchasesToDeviceParams{
		TargetDeviceID: sql.NullInt64{Int64: targetDeviceID, Valid: true},
		SourceDeviceID: sql.NullInt64{Int64: sourceDeviceID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to transfer purchases: %w", err)
	}

	// 3. Transfer fingerprints from source to target
	err = qtx.TransferFingerprintsToDevice(ctx, queries.TransferFingerprintsToDeviceParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		return fmt.Errorf("failed to transfer fingerprints: %w", err)
	}

	// 4. Transfer MAC address records from source to target
	err = qtx.DeleteConflictingMacsBeforeTransfer(ctx, queries.DeleteConflictingMacsBeforeTransferParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete conflicting MAC addresses: %w", err)
	}
	err = qtx.TransferMacs(ctx, queries.TransferMacsParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
	if err != nil {
		return fmt.Errorf("failed to transfer MAC addresses: %w", err)
	}

	// 5. Merge wallets - transfer balance and transactions
	sourceWallet, err := qtx.FindWalletByDeviceId(ctx, sourceDeviceID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to find source wallet: %w", err)
		}
	} else {
		targetWallet, err := qtx.FindWalletByDeviceId(ctx, targetDeviceID)
		if err != nil {
			return fmt.Errorf("failed to find target wallet: %w", err)
		}

		// Transfer wallet transactions from source to target wallet
		err = qtx.TransferWalletTransactions(ctx, queries.TransferWalletTransactionsParams{
			TargetWalletID: targetWallet.ID,
			SourceWalletID: sourceWallet.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to transfer wallet transactions: %w", err)
		}

		// Add source wallet balance to target wallet
		if sourceWallet.Balance > 0 {
			err = qtx.AddToWalletBalance(ctx, queries.AddToWalletBalanceParams{
				Amount:   sourceWallet.Balance,
				DeviceID: targetDeviceID,
			})
			if err != nil {
				return fmt.Errorf("failed to add wallet balance: %w", err)
			}
		}

		// Delete source wallet (transactions already transferred)
		err = qtx.DeleteWalletByDeviceId(ctx, sourceDeviceID)
		if err != nil {
			return fmt.Errorf("failed to delete source wallet: %w", err)
		}
	}

	// 6. Delete the source device
	err = qtx.DeleteDevice(ctx, sourceDeviceID)
	if err != nil {
		return fmt.Errorf("failed to delete source device: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// VACUUM is intentionally omitted here. The database operates in WAL mode,
	// which reclaims space automatically via periodic checkpointing. Calling
	// VACUUM in WAL mode forces a full database rewrite and blocks all concurrent
	// readers and writers for its duration — unacceptable in a merge job that may
	// process many pairs per run.

	return nil
}
