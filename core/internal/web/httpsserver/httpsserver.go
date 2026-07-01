package httpsserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"core/utils/config"
	"core/utils/env"
	sdkutils "github.com/flarewifi/sdk-utils"
	"github.com/gorilla/mux"
)

var (
	certDir  = filepath.Join(sdkutils.PathDataDir, "storage", "certs")
	certFile = filepath.Join(certDir, "server.crt")
	keyFile  = filepath.Join(certDir, "server.key")
	// Seed cert+key embedded into the release by the software-release build
	// (EmbedPortalCertificate). On first boot, before the cloud-sync fetch job has
	// run, these let the portal serve the real cloud-issued cert instead of a
	// self-signed one. Absent or expired => fall back to self-signed.
	seedCertFile = filepath.Join(sdkutils.PathDefaultsDir, "certs", "server.crt")
	seedKeyFile  = filepath.Join(sdkutils.PathDefaultsDir, "certs", "server.key")
	// Renew certificates when they expire within this duration
	renewalThreshold = 30 * 24 * time.Hour // 30 days

	// HTTPS server state management
	httpsServer        *http.Server
	httpsServerMu      sync.Mutex
	httpsServerRunning bool
	certRenewalStop    chan struct{}
	currentRouter      *mux.Router
)

// isCertificateExpired checks if the certificate is expired or needs renewal
func isCertificateExpired() (bool, error) {
	if !sdkutils.FsExists(certFile) {
		return true, nil
	}

	certData, err := ioutil.ReadFile(certFile)
	if err != nil {
		return true, err
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		return true, errors.New("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true, err
	}

	// Check if certificate expires within the renewal threshold
	return time.Until(cert.NotAfter) < renewalThreshold, nil
}

// currentCertIsSelfSigned reports whether the cert currently on disk is one of our
// self-signed fallbacks (a self-signed cert has Issuer == Subject), as opposed to a
// cloud-issued cert (signed by a real CA, so Issuer != Subject). A missing or
// unparseable cert returns false so the caller falls back to the plain expiry check.
// This is what lets ensureTLSCertificates refuse to keep a never-expiring self-signed
// fallback in place of the embedded cloud cert on a portal-domain build.
func currentCertIsSelfSigned() bool {
	if !sdkutils.FsExists(certFile) {
		return false
	}
	certData, err := os.ReadFile(certFile)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(certData)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return bytes.Equal(cert.RawIssuer, cert.RawSubject)
}

// ensureTLSCertificates checks if TLS certificates exist and are valid, generates them if needed
func ensureTLSCertificates() error {
	// Check if certificates need renewal
	expired, err := isCertificateExpired()
	if err != nil {
		return fmt.Errorf("error checking certificate expiration: %v", err)
	}

	// Keep a non-expired cert that's already on disk — EXCEPT a self-signed fallback
	// on a build that is supposed to serve the cloud-issued cert (HasCustomDomain).
	// The fallback is minted with a 10-year validity, so it never trips
	// isCertificateExpired; once written it would otherwise shadow the embedded cloud
	// cert FOREVER. That is exactly what strands a device which self-signed on an
	// earlier boot (e.g. before NTP synced, or before the image shipped a seed cert) —
	// including across a reflash, since certFile lives on the persistent data partition
	// (data/storage/certs), not in the re-imaged app/. Falling through here lets the
	// seed step below replace that fallback with the real cert.
	if !expired && !(config.HasCustomDomain() && currentCertIsSelfSigned()) {
		return nil
	}

	// Ensure the certs directory exists
	if err := sdkutils.FsEnsureDir(certDir); err != nil {
		return err
	}

	// The release-embedded seed cert is the cloud-issued cert for this build's
	// portal domain. Only use it when the build has a portal domain (staging/prod):
	// it is the first-boot head start so a fresh install serves the real portal cert
	// immediately (the runtime cloud-sync fetch keeps it current afterwards). With
	// NO portal domain (dev/devkit) there is no host for that cert to match, so skip
	// seeding and generate a self-signed cert below.
	if config.HasCustomDomain() && seedFromDefaults() {
		return nil
	}

	// We can land here with a still-valid (non-expired) self-signed cert when the
	// build has a portal domain but no seed was usable yet (the cloud hasn't issued a
	// portal cert). Keep the existing fallback rather than churning a fresh self-signed
	// keypair on every call; the runtime cloud-sync fetch installs the real cert later.
	if !expired {
		return nil
	}

	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Flarewifi"},
			CommonName:   "Flarewifi Router",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}

// seedFromDefaults installs the release-embedded portal cert from defaults/certs into
// the live certs dir when it exists, the key matches the cert, and the cert is within
// its validity window. It returns true on a successful install. A missing, mismatched,
// or expired seed returns false so the caller falls back to a self-signed cert (the
// runtime cloud-sync fetch then supplies the current cert). Best-effort: it never
// errors out the boot path.
func seedFromDefaults() bool {
	if !sdkutils.FsExists(seedCertFile) || !sdkutils.FsExists(seedKeyFile) {
		return false
	}

	certPEM, err := os.ReadFile(seedCertFile)
	if err != nil {
		return false
	}
	keyPEM, err := os.ReadFile(seedKeyFile)
	if err != nil {
		return false
	}

	// The key must actually match the cert. We deliberately do NOT reject the seed on
	// its validity window (NotBefore/NotAfter): a fresh router boots with an unsynced
	// clock (no RTC, NTP hasn't run yet) that is frequently BEHIND the cert's
	// NotBefore, so a perfectly good cloud cert would look "not yet valid" and be
	// dropped — the original reason the machine self-signed despite shipping a valid
	// cert. A domain-matching cloud cert is strictly better than a self-signed one even
	// when the local clock disagrees about validity, and the runtime cloud-sync fetch
	// (jobs.performPortalCertFetch) replaces it with a current cert once the clock and
	// network are up.
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return false
	}

	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		return false
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return false
	}
	return true
}

// startCertificateRenewalChecker runs periodic checks for certificate renewal
// Accepts a stop channel for graceful shutdown
func startCertificateRenewalChecker(stop chan struct{}) {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			expired, err := isCertificateExpired()
			if err != nil {
				continue
			}

			if expired {
				_ = ensureTLSCertificates()
			}
		}
	}
}

// StartHTTPSServer starts the HTTPS server if enabled in config
func StartHTTPSServer(r *mux.Router) error {
	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()

	// Store the router for potential restarts
	currentRouter = r

	if httpsServerRunning {
		return nil
	}

	// The HTTPS server always runs so both the captive portal and the admin pages
	// are served over a valid (cloud-issued) certificate. HTTP->HTTPS upgrade is
	// enforced globally by middlewares.ForceHTTPS.
	addr := fmt.Sprintf(":%d", env.HTTPS_PORT)

	// Ensure TLS certificates exist
	if err := ensureTLSCertificates(); err != nil {
		return fmt.Errorf("failed to ensure TLS certificates: %w", err)
	}

	srv := &http.Server{
		Handler: withAltSvcClear(r),
		Addr:    addr,
	}

	httpsServer = srv
	httpsServerRunning = true
	certRenewalStop = make(chan struct{})

	// Start periodic certificate renewal checker
	go startCertificateRenewalChecker(certRenewalStop)

	go func() {
		_ = srv.ListenAndServeTLS(certFile, keyFile)
	}()

	return nil
}

// withAltSvcClear emits `Alt-Svc: clear` on every HTTPS response. The machine
// serves TLS over TCP only (HTTP/1.1 + H2) — there is NO QUIC/HTTP-3 listener.
// But the cloud zone advertises h3 on its Cloudflare-proxied hosts (e.g.
// `alt-svc: h3=":443"`), which can seed a browser's HTTP/3 broker with an h3
// entry for the captive-portal hostname as well. The browser then attempts QUIC
// against the machine's UDP :443 (nothing listens there) and the page fails with
// ERR_QUIC_PROTOCOL_ERROR. `Alt-Svc: clear` tells the browser to drop any cached
// alternative-service (h3) for this origin so it stays on TCP. Set on every
// response (before the wrapped handler writes) so it covers redirects/errors too.
func withAltSvcClear(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Alt-Svc", "clear")
		next.ServeHTTP(w, r)
	})
}

// StopHTTPSServer gracefully stops the HTTPS server
func StopHTTPSServer() {
	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()

	if !httpsServerRunning || httpsServer == nil {
		return
	}

	// Stop certificate renewal checker
	close(certRenewalStop)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpsServer.Shutdown(ctx)

	httpsServer = nil
	httpsServerRunning = false
}

// IsHTTPSServerRunning returns true if the HTTPS server is currently running
func IsHTTPSServerRunning() bool {
	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()
	return httpsServerRunning
}

// GetCurrentRouter returns the router that was used to start the server
func GetCurrentRouter() *mux.Router {
	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()
	return currentRouter
}

// CurrentCertFingerprint returns the sha256 (hex) of the certificate currently
// on disk, or "" if none exists. It matches the fingerprint the cloud computes
// over the same PEM bytes, so the device can ask whether a newer cert exists.
func CurrentCertFingerprint() string {
	if !sdkutils.FsExists(certFile) {
		return ""
	}
	b, err := os.ReadFile(certFile)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// InstallCertificate writes a cloud-issued certificate + key to disk and, if the
// HTTPS server is running, restarts it so the new material takes effect. Used by
// the portal-cert fetch job after pulling a changed cert from the cloud.
func InstallCertificate(certPEM, keyPEM []byte) error {
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return errors.New("install certificate: empty cert or key")
	}

	if err := sdkutils.FsEnsureDir(certDir); err != nil {
		return fmt.Errorf("ensure certs dir: %w", err)
	}
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	// Snapshot running state under the lock, then (re)start outside it (Stop/Start
	// take the same mutex, so holding it here would deadlock).
	httpsServerMu.Lock()
	running := httpsServerRunning
	router := currentRouter
	httpsServerMu.Unlock()

	// No router captured yet means StartHTTPSServer has never been attempted;
	// the next start will pick up the cert from disk.
	if router == nil {
		return nil
	}

	// If a prior start failed (e.g. certs dir wasn't writable) the server is not
	// running but the router is known — start it now that a cert exists.
	if running {
		StopHTTPSServer()
	}
	return StartHTTPSServer(router)
}
