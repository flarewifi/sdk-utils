package jobs

import (
	"context"
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/internal/web/httpsserver"
	"core/utils/config"
	"time"

	sdkapi "sdk/api"
)

func StartPortalCertScheduler(scheduler sdkapi.ISchedulerApi) error {
	return scheduler.Go("portal-cert", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(PortalCertInitialDelay):
		}
		performPortalCertFetch()

		ticker := time.NewTicker(PortalCertInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				performPortalCertFetch()
			}
		}
	})
}

func performPortalCertFetch() {
	// No portal domain (dev/devkit) => the machine serves a self-signed cert; there
	// is no cloud-issued portal cert to fetch (and fetching one would clobber the
	// self-signed cert). Staging/prod have a portal domain and do fetch.
	if !config.HasCustomDomain() {
		return
	}

	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchPortalCertificate(ctx, &rpc_flarewifi_v3.FetchPortalCertificateRequest{
		MachineId:       machineID,
		HaveFingerprint: httpsserver.CurrentCertFingerprint(),
	})
	if err != nil {
		return
	}

	if !resp.GetSuccess() {
		return
	}

	if !resp.GetChanged() {
		return
	}

	if err := httpsserver.InstallCertificate([]byte(resp.GetCertPem()), []byte(resp.GetKeyPem())); err != nil {
		return
	}
}
