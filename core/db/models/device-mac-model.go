package models

import (
	"context"
	"database/sql"
	"log"

	"core/db"
	"core/db/queries"
)

type DeviceMacModel struct {
	db     *db.Database
	models *Models
}

func NewDeviceMacModel(database *db.Database, mdls *Models) *DeviceMacModel {
	return &DeviceMacModel{db: database, models: mdls}
}

// RecordMacAddress records a MAC address for a device with limit enforcement (100 MAC limit)
func (self *DeviceMacModel) RecordMacAddress(ctx context.Context, deviceID int64, macAddress string) error {
	// Check if MAC already exists
	existing, err := self.CheckExisting(ctx, deviceID, macAddress)
	if err != nil {
		return err
	}

	if existing != nil {
		// MAC exists - update last_seen and set as current
		log.Printf("[ClientDeviceMac] MAC %s already exists for device %d, updating", macAddress, deviceID)
		if err := self.SetAsCurrent(ctx, existing.ID, deviceID); err != nil {
			return err
		}
		return nil
	}

	// Check MAC limit (100 per device)
	count, err := self.db.Queries.GetMacCountByDeviceID(ctx, deviceID)
	if err != nil {
		return err
	}

	if count >= 100 {
		log.Printf("[ClientDeviceMac] WARN: Device %d has %d MACs (limit reached), deleting oldest inactive MAC", deviceID, count)
		if err := self.db.Queries.DeleteOldestInactiveMac(ctx, deviceID); err != nil {
			log.Printf("[ClientDeviceMac] ERROR: Failed to delete oldest MAC: %v", err)
			// Continue anyway - don't block on cleanup
		}
	}

	// Create new MAC record
	log.Printf("[ClientDeviceMac] Creating new MAC record for device %d: %s", deviceID, macAddress)
	_, err = self.db.Queries.CreateDeviceMac(ctx, queries.CreateDeviceMacParams{
		DeviceID:   deviceID,
		MacAddress: macAddress,
		IsCurrent:  false, // Will be set by SetAsCurrent
	})
	if err != nil {
		return err
	}

	// Get the newly created record
	newMac, err := self.CheckExisting(ctx, deviceID, macAddress)
	if err != nil {
		return err
	}

	// Set as current
	return self.SetAsCurrent(ctx, newMac.ID, deviceID)
}

// Create creates a new MAC address record
func (self *DeviceMacModel) Create(ctx context.Context, params queries.CreateDeviceMacParams) (int64, error) {
	return self.db.Queries.CreateDeviceMac(ctx, params)
}

// FindByDeviceID returns all MAC addresses for a device
func (self *DeviceMacModel) FindByDeviceID(ctx context.Context, deviceID int64) ([]queries.DeviceMac, error) {
	return self.db.Queries.FindMacsByDeviceID(ctx, deviceID)
}

// FindCurrentMac returns the current active MAC for a device
func (self *DeviceMacModel) FindCurrentMac(ctx context.Context, deviceID int64) (*queries.DeviceMac, error) {
	row, err := self.db.Queries.FindCurrentMacByDeviceID(ctx, deviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// CheckExisting checks if a MAC already exists for device
func (self *DeviceMacModel) CheckExisting(ctx context.Context, deviceID int64, macAddress string) (*queries.DeviceMac, error) {
	row, err := self.db.Queries.CheckExistingMac(ctx, queries.CheckExistingMacParams{
		DeviceID:   deviceID,
		MacAddress: macAddress,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// SetAsCurrent marks a MAC as current (and unmarks all others for device)
func (self *DeviceMacModel) SetAsCurrent(ctx context.Context, id int64, deviceID int64) error {
	return self.db.Queries.SetMacAsCurrent(ctx, queries.SetMacAsCurrentParams{
		ID:       id,
		DeviceID: deviceID,
	})
}

// UpdateLastSeen updates the last_seen_at timestamp
func (self *DeviceMacModel) UpdateLastSeen(ctx context.Context, id int64) error {
	return self.db.Queries.UpdateMacLastSeen(ctx, id)
}

// TransferMacs transfers all MAC records from source to target device
func (self *DeviceMacModel) TransferMacs(ctx context.Context, targetDeviceID, sourceDeviceID int64) error {
	return self.db.Queries.TransferMacs(ctx, queries.TransferMacsParams{
		TargetDeviceID: targetDeviceID,
		SourceDeviceID: sourceDeviceID,
	})
}

// DeleteNonCurrent deletes all non-current MAC address records across all devices.
func (self *DeviceMacModel) DeleteNonCurrent(ctx context.Context) error {
	return self.db.Queries.DeleteNonCurrentMacs(ctx)
}

// UnsetCurrentMac unmarks a specific MAC as current on a specific device.
// Used when a MAC collision is detected but merge is rejected.
func (self *DeviceMacModel) UnsetCurrentMac(ctx context.Context, deviceID int64, macAddress string) error {
	_, err := self.db.DB.ExecContext(ctx,
		"UPDATE device_macs SET is_current = FALSE WHERE device_id = ? AND mac_address = ? AND is_current = TRUE",
		deviceID, macAddress)
	return err
}

// FindDeviceByMac finds a device ID by MAC address (current MACs only)
func (self *DeviceMacModel) FindDeviceByMac(ctx context.Context, macAddress string) (int64, error) {
	deviceID, err := self.db.Queries.FindDeviceByMacAddress(ctx, macAddress)
	if err != nil {
		return 0, err
	}
	return deviceID, nil
}

// FindDeviceByAnyMac finds a device ID by ANY MAC address in history (not just current)
// Returns the device that most recently used this MAC
func (self *DeviceMacModel) FindDeviceByAnyMac(ctx context.Context, macAddress string) (int64, error) {
	deviceID, err := self.db.Queries.FindDeviceByAnyMacAddress(ctx, macAddress)
	if err != nil {
		return 0, err
	}
	return deviceID, nil
}
