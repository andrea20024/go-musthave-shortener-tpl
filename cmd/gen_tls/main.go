// Package main is a utility for generating self-signed TLS certificates.
//
// Generates an RSA 2048-bit key pair and a self-signed X.509 certificate
// valid for "localhost" and "127.0.0.1" with Subject Alternative Names.
//
// Usage:
//
//	go run ./cmd/gen_tls/...
//	go run ./cmd/gen_tls/... -o certs
//	go run ./cmd/gen_tls/... -days 365
//
// Output files:
//
//	certs/server.crt  — PEM certificate
//	certs/server.key  — PEM private key
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

var (
	outputDir = flag.String("o", "certs", "output directory for certificate files")
	days      = flag.Int("days", 365, "certificate validity period in days")
)

func main() {
	flag.Parse()

	if *days <= 0 {
		fmt.Fprintf(os.Stderr, "error: days must be positive\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	certPath := filepath.Join(*outputDir, "server.crt")
	keyPath := filepath.Join(*outputDir, "server.key")

	if err := GenerateSelfSignedCert(certPath, keyPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("certificate: %s\n", certPath)
	fmt.Printf("private key: %s\n", keyPath)
}

// GenerateSelfSignedCert creates a self-signed TLS certificate and key for localhost.
//
// The generated certificate uses RSA 2048-bit key, is valid for 1 year, and
// includes SAN for "localhost" and "127.0.0.1". Files are written as PEM format.
//
// Returns an error if key generation, certificate creation, or file writing fails.
func GenerateSelfSignedCert(certFile, keyFile string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("create cert: %w", err)
	}

	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), 0644); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}), 0644); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	return nil
}
