package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	err := GenerateSelfSignedCert(certFile, keyFile)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Check files exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Fatal("certificate file not created")
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Fatal("key file not created")
	}

	// Verify certificate can be parsed
	certData, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("failed to read cert file: %v", err)
	}
	block, _ := pem.Decode(certData)
	if block == nil {
		t.Fatal("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Verify certificate properties
	if cert.Subject.CommonName != "localhost" {
		t.Errorf("expected CommonName=localhost, got %s", cert.Subject.CommonName)
	}
}
