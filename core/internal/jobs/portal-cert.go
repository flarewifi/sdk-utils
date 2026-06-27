package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/internal/web/httpsserver"
	"core/utils/config"
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
	// No custom_domain => the machine serves a self-signed cert; there is no
	// cloud-issued portal cert to fetch (and fetching one would clobber the
	// self-signed cert). Holds in dev, staging, and prod alike.
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
