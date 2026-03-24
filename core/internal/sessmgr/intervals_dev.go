//go:build dev

package sessmgr

import "time"

// Dev mode: short intervals for fast iteration and testing.

var (
	// BatchSaveInterval controls how often the batch save loop snapshots
	// time consumption for all running sessions and persists them to the
	// database in a single transaction.
	BatchSaveInterval = 30 * time.Second
)
