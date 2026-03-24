//go:build !dev

package sessmgr

import "time"

// Production mode: longer intervals to conserve CPU and disk I/O on embedded devices.

var (
	// BatchSaveInterval controls how often the batch save loop snapshots
	// time consumption for all running sessions and persists them to the
	// database in a single transaction.
	BatchSaveInterval = 1 * time.Minute
)
