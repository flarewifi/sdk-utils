package jobs

import (
	"context"
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"time"

	sdkapi "sdk/api"
)

// StartMachinePingScheduler pings the cloud every hour (after an initial
// settle delay) so the machine's online status stays fresh.
func StartMachinePingScheduler(scheduler sdkapi.ISchedulerApi) error {
	return scheduler.Go("machine-ping", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(MachinePingInitialDelay):
		}
		performMachinePing()

		ticker := time.NewTicker(MachinePingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				performMachinePing()
			}
		}
	})
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
