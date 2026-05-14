package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"log"
	"time"
)

// StartMachinePingScheduler starts a background goroutine that pings
// the server to update last_ping_at for online status.
// In dev mode: pings every 5 seconds. In prod: pings every hour.
func StartMachinePingScheduler() {
	go func() {
		if MachinePingInterval < time.Hour {
			log.Printf("[MachinePing] DEV MODE: Pinging every %v", MachinePingInterval)
		} else {
			log.Println("[MachinePing] Scheduler started - will ping every hour")
		}

		// Initial ping on startup (with delay to allow system to stabilize)
		time.Sleep(MachinePingInitialDelay)
		performMachinePing()

		// Then ping at configured interval
		ticker := time.NewTicker(MachinePingInterval)
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
