package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"time"
)

func StartMachinePingScheduler() {
	go func() {
		time.Sleep(MachinePingInitialDelay)
		performMachinePing()

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
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()

	_, err := srv.MachinePing(ctx, &rpc_flarewifi_v3.MachinePingRequest{
		MachineId: machineID,
	})

	if err != nil {
		return
	}
}
