package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/internal/web/httpsserver"
	"time"
)

func StartPortalCertScheduler() {
	go func() {
		time.Sleep(PortalCertInitialDelay)
		performPortalCertFetch()

		ticker := time.NewTicker(PortalCertInterval)
		defer ticker.Stop()
		for range ticker.C {
			performPortalCertFetch()
		}
	}()
}

func performPortalCertFetch() {
	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchPortalCertificate(ctx, &rpc_flarewifi_v2.FetchPortalCertificateRequest{
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
