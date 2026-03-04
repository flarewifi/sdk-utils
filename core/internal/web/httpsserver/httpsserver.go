//go:build !dev

package httpsserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"core/utils/config"
	"core/utils/env"
	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/gorilla/mux"
)

var (
	certDir  = filepath.Join(sdkutils.PathDataDir, "storage", "certs")
	certFile = filepath.Join(certDir, "server.crt")
	keyFile  = filepath.Join(certDir, "server.key")
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

// ensureTLSCertificates checks if TLS certificates exist and are valid, generates them if needed
func ensureTLSCertificates() error {
	// Check if certificates need renewal
	expired, err := isCertificateExpired()
	if err != nil {
		return fmt.Errorf("error checking certificate expiration: %v", err)
	}

	if !expired {
		log.Println("TLS certificates are valid at", certDir)
		return nil
	}

	log.Println("TLS certificates expired or missing, generating new ones...")

	log.Println("Generating self-signed TLS certificates...")

	// Ensure the certs directory exists
	if err := sdkutils.FsEnsureDir(certDir); err != nil {
		return err
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
			Organization: []string{"FlareWifi"},
			CommonName:   "FlareWifi Router",
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

	log.Println("Generated certificate:", certFile)

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

	log.Println("Generated private key:", keyFile)
	log.Println("Self-signed TLS certificates created successfully")

	return nil
}

// startCertificateRenewalChecker runs periodic checks for certificate renewal
// Accepts a stop channel for graceful shutdown
func startCertificateRenewalChecker(stop chan struct{}) {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Println("Certificate renewal checker stopped")
			return
		case <-ticker.C:
			expired, err := isCertificateExpired()
			if err != nil {
				log.Printf("Error checking certificate expiration: %v\n", err)
				continue
			}

			if expired {
				log.Println("Certificate approaching expiration, renewing...")
				if err := ensureTLSCertificates(); err != nil {
					log.Printf("Error renewing certificate: %v\n", err)
				} else {
					log.Println("Certificate renewed successfully")
				}
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
		log.Println("HTTPS server is already running")
		return nil
	}

	// Check if HTTPS is enabled in config
	cfg, err := config.GetCachedAppConfig()
	if err != nil {
		log.Printf("Error reading application config: %v\n", err)
		return err
	}

	if !cfg.AdminWebHttps {
		log.Println("Admin HTTPS is disabled in config, skipping HTTPS server")
		return nil
	}

	addr := fmt.Sprintf(":%d", env.HTTPS_PORT)
	log.Println("Starting HTTPS server on port", addr)

	// Ensure TLS certificates exist
	if err := ensureTLSCertificates(); err != nil {
		log.Printf("Error ensuring TLS certificates: %v\n", err)
		return fmt.Errorf("failed to ensure TLS certificates: %w", err)
	}

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	httpsServer = srv
	httpsServerRunning = true
	certRenewalStop = make(chan struct{})

	// Start periodic certificate renewal checker
	go startCertificateRenewalChecker(certRenewalStop)

	go func() {
		err := srv.ListenAndServeTLS(certFile, keyFile)
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Printf("Error starting HTTPS server: %v\n", err)
		}
	}()

	return nil
}

// StopHTTPSServer gracefully stops the HTTPS server
func StopHTTPSServer() {
	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()

	if !httpsServerRunning || httpsServer == nil {
		log.Println("HTTPS server is not running")
		return
	}

	log.Println("Stopping HTTPS server...")

	// Stop certificate renewal checker
	close(certRenewalStop)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpsServer.Shutdown(ctx); err != nil {
		log.Printf("Error stopping HTTPS server: %v\n", err)
	} else {
		log.Println("HTTPS server stopped successfully")
	}

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
