package sdkconnmgr

import (
	"context"
	"time"
)

type SessionData struct {
	Provider       string
	Type           uint8
	TimeSecs       uint
	DataMb         float64
	TimeCons       uint
	DataCons       float64
	StartedAt      *time.Time
	ExpDays        *uint
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
	CreatedAt      time.Time
}

type ISessionSource interface {

	// Return the session data.
	Data() SessionData

	// Save data to the source, e.g. database.
	Save(context.Context, SessionData) error

	// Reload data from the source, e.g. database.
	Reload(context.Context) (SessionData, error)
}
