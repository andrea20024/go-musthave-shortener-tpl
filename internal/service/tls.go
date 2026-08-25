// Package service provides business logic and utility functions for the URL
// shortener application.
//
// It contains service layers for core operations such as URL storage,
// certificate generation, and other domain-specific logic.
package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

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
