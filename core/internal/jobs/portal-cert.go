package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/internal/web/httpsserver"
	"log"
	"time"
)

// StartPortalCertScheduler periodically fetches the shared captive-portal TLS
// certificate from the cloud (over the flarewifi v2 RPC) and installs it locally,
// hot-reloading the HTTPS server when it changes. The cloud holds the shared key,
// so the device just writes what it receives.
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
		log.Println("[PortalCert] Machine ID not available, skipping fetch")
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchPortalCertificate(ctx, &rpc_flarewifi_v2.FetchPortalCertificateRequest{
		MachineId:       machineID,
		HaveFingerprint: httpsserver.CurrentCertFingerprint(),
	})
	if err != nil {
		log.Printf("[PortalCert] Fetch failed: %v", err)
		return
	}

	if !resp.GetSuccess() {
		log.Printf("[PortalCert] Cloud certificate not available yet: %s", resp.GetErrorMessage())
		return
	}

	if !resp.GetChanged() {
		log.Println("[PortalCert] Certificate already up to date")
		return
	}

	if err := httpsserver.InstallCertificate([]byte(resp.GetCertPem()), []byte(resp.GetKeyPem())); err != nil {
		log.Printf("[PortalCert] Failed to install certificate: %v", err)
		return
	}

	log.Printf("[PortalCert] Installed new certificate for %s (fingerprint %s)", resp.GetDomain(), resp.GetFingerprint())
}
