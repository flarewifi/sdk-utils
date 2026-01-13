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
	rows, err := self.db.Queries.FindFingerprintsByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Convert rows to DeviceFingerprint structs
	result := make([]queries.DeviceFingerprint, len(rows))
	for i, row := range rows {
		result[i] = queries.DeviceFingerprint{
			ID:               row.ID,
			DeviceID:         row.DeviceID,
			FingerprintHash:  row.FingerprintHash,
			UserAgent:        row.UserAgent,
			BrowserName:      row.BrowserName,
			OsFamily:         row.OsFamily,
			ScreenResolution: row.ScreenResolution,
			Language:         row.Language,
			Timezone:         row.Timezone,
			IsCna:            row.IsCna,
			CreatedAt:        row.CreatedAt,
			LastSeenAt:       row.LastSeenAt,
		}
	}
	return result, nil
}

// CheckExactMatch checks if a fingerprint with exact hash exists for device
func (self *DeviceFingerprintModel) CheckExactMatch(ctx context.Context, deviceID int64, hash string) (*queries.DeviceFingerprint, error) {
	row, err := self.db.Queries.CheckFingerprintExactMatch(ctx, queries.CheckFingerprintExactMatchParams{
		DeviceID:        deviceID,
		FingerprintHash: hash,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Convert row to DeviceFingerprint struct
	fp := &queries.DeviceFingerprint{
		ID:               row.ID,
		DeviceID:         row.DeviceID,
		FingerprintHash:  row.FingerprintHash,
		UserAgent:        row.UserAgent,
		BrowserName:      row.BrowserName,
		OsFamily:         row.OsFamily,
		ScreenResolution: row.ScreenResolution,
		Language:         row.Language,
		Timezone:         row.Timezone,
		IsCna:            row.IsCna,
		CreatedAt:        row.CreatedAt,
		LastSeenAt:       row.LastSeenAt,
	}
	return fp, nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a fingerprint
func (self *DeviceFingerprintModel) UpdateLastSeen(ctx context.Context, id int64) error {
	return self.db.Queries.UpdateFingerprintLastSeen(ctx, id)
}

// DeleteOldFingerprints removes fingerprints older than 6 months
func (self *DeviceFingerprintModel) DeleteOldFingerprints(ctx context.Context) error {
	return self.db.Queries.DeleteOldFingerprints(ctx)
}
