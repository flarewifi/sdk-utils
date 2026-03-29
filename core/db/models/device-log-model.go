package models

import (
	"context"
	"encoding/json"
	"log"

	"core/db"
	"core/db/queries"
)

const DeviceLogsPerPage = 20

type DeviceLogModel struct {
	db     *db.Database
	models *Models
}

type DeviceLogPaginateResult struct {
	Logs       []queries.DeviceLog
	TotalCount int64
	Page       int
	PerPage    int
	TotalPages int
}

func NewDeviceLogModel(database *db.Database, mdls *Models) *DeviceLogModel {
	return &DeviceLogModel{db: database, models: mdls}
}

// Create creates a new device log entry with JSON metadata
func (self *DeviceLogModel) Create(ctx context.Context, deviceID int64, message string, metadata map[string]interface{}) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("[DeviceLogModel.Create] WARN: Failed to marshal metadata: %v", err)
		metaJSON = []byte("{}")
	}

	_, err = self.db.Queries.CreateDeviceLog(ctx, queries.CreateDeviceLogParams{
		DeviceID: deviceID,
		Message:  message,
		Metadata: string(metaJSON),
	})
	return err
}

// FindByDeviceIDPaginated returns paginated logs for a device
func (self *DeviceLogModel) FindByDeviceIDPaginated(ctx context.Context, deviceID int64, page int) (*DeviceLogPaginateResult, error) {
	if page < 1 {
		page = 1
	}

	totalCount, err := self.db.Queries.CountDeviceLogsByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / DeviceLogsPerPage
	if int(totalCount)%DeviceLogsPerPage > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	offset := (page - 1) * DeviceLogsPerPage
	logs, err := self.db.Queries.FindDeviceLogsByDeviceID(ctx, queries.FindDeviceLogsByDeviceIDParams{
		DeviceID:   deviceID,
		PageLimit:  int64(DeviceLogsPerPage),
		PageOffset: int64(offset),
	})
	if err != nil {
		return nil, err
	}

	return &DeviceLogPaginateResult{
		Logs:       logs,
		TotalCount: totalCount,
		Page:       page,
		PerPage:    DeviceLogsPerPage,
		TotalPages: totalPages,
	}, nil
}

// DeleteByDeviceID removes all logs for a device
func (self *DeviceLogModel) DeleteByDeviceID(ctx context.Context, deviceID int64) error {
	return self.db.Queries.DeleteDeviceLogsByDeviceID(ctx, deviceID)
}

// DeleteOld removes logs older than 90 days
func (self *DeviceLogModel) DeleteOld(ctx context.Context) error {
	return self.db.Queries.DeleteOldDeviceLogs(ctx)
}
