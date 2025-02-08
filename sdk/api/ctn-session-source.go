/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	SessionTypeTime       string = "time"
	SessionTypeData       string = "data"
	SessionTypeTimeOrData string = "time-or-data"
)

type SessionData struct {
	Id             pgtype.UUID
	Provider       string
	Type           string
	TimeSecs       int
	DataMb         float64
	TimeCons       int
	DataCons       float64
	StartedAt      *time.Time
	ExpDays        *int
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
