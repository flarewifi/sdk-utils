package boot

import (
	"context"
	"log"

	"core/internal/api"
)

// BackfillDeviceUUIDs generates UUIDs for devices that have empty UUID fields
func BackfillDeviceUUIDs(g *api.CoreGlobals) {
	ctx := context.Background()
	err := g.Models.Device().BackfillEmptyUUIDs(ctx)
	if err != nil {
		log.Printf("Error backfilling device UUIDs: %v", err)
	}
}
