package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc_flarewifi_v2"
	"log"
	"time"
)

// StartMachinePingScheduler starts a background goroutine that pings
// the server every hour to update last_ping_at for online status
func StartMachinePingScheduler() {
	go func() {
		// Initial ping on startup (with small delay to allow system to stabilize)
		time.Sleep(30 * time.Second)
		performMachinePing()

		// Then ping every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			performMachinePing()
		}
	}()
}

func performMachinePing() {
	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		log.Println("[MachinePing] Machine ID not available, skipping ping")
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()

	_, err := srv.MachinePing(ctx, &rpc_flarewifi_v2.MachinePingRequest{
		MachineId: machineID,
	})

	if err != nil {
		log.Printf("[MachinePing] Failed to ping server: %v", err)
		return
	}

	log.Printf("[MachinePing] Successfully pinged server for machine: %s", machineID)
}
