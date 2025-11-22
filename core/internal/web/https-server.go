package web

import (
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
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/gorilla/mux"
	"tools/env"
)

var (
	certDir  = filepath.Join(sdkutils.PathDataDir, "system", "certs")
	certFile = filepath.Join(certDir, "server.crt")
	keyFile  = filepath.Join(certDir, "server.key")
	// Renew certificates when they expire within this duration
	renewalThreshold = 30 * 24 * time.Hour // 30 days
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
func startCertificateRenewalChecker() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
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

func StartHTTPSServer(r *mux.Router) {
	addr := fmt.Sprintf(":%d", env.HTTPS_PORT)
	log.Println("Starting HTTPS server on port", addr)

	// Ensure TLS certificates exist
	if err := ensureTLSCertificates(); err != nil {
		log.Printf("Error ensuring TLS certificates: %v\n", err)
		return
	}

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	// Start periodic certificate renewal checker
	go startCertificateRenewalChecker()

	go func() {
		err := srv.ListenAndServeTLS(certFile, keyFile)
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Printf("Error starting HTTPS server: %v\n", err)
		}
	}()
}
