package models

import (
	"context"
	"database/sql"

	"core/db"
	"core/db/queries"
)

type DeviceFingerprintModel struct {
	db     *db.Database
	models *Models
}

func NewDeviceFingerprintModel(database *db.Database, mdls *Models) *DeviceFingerprintModel {
	return &DeviceFingerprintModel{db: database, models: mdls}
}

// Create creates a new device fingerprint record
func (self *DeviceFingerprintModel) Create(ctx context.Context, params queries.CreateDeviceFingerprintParams) (int64, error) {
	return self.db.Queries.CreateDeviceFingerprint(ctx, params)
}

// FindByDeviceID returns all fingerprints for a device (within 6 months)
func (self *DeviceFingerprintModel) FindByDeviceID(ctx context.Context, deviceID int64) ([]queries.DeviceFingerprint, error) {
	return self.db.Queries.FindFingerprintsByDeviceID(ctx, deviceID)
}

// CheckExactMatch checks if a fingerprint with exact hash exists for device
func (self *DeviceFingerprintModel) CheckExactMatch(ctx context.Context, deviceID int64, hash string) (*queries.DeviceFingerprint, error) {
	fp, err := self.db.Queries.CheckFingerprintExactMatch(ctx, queries.CheckFingerprintExactMatchParams{
		DeviceID:        deviceID,
		FingerprintHash: hash,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &fp, nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a fingerprint
func (self *DeviceFingerprintModel) UpdateLastSeen(ctx context.Context, id int64) error {
	return self.db.Queries.UpdateFingerprintLastSeen(ctx, id)
}

// DeleteOldFingerprints removes fingerprints older than 6 months
func (self *DeviceFingerprintModel) DeleteOldFingerprints(ctx context.Context) error {
	return self.db.Queries.DeleteOldFingerprints(ctx)
}
