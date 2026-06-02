// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSCreds_InsecureFallback(t *testing.T) {
	creds, err := tlsCreds("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Info().SecurityProtocol != "insecure" {
		t.Errorf("expected insecure protocol, got %q", creds.Info().SecurityProtocol)
	}
}

func TestTLSCreds_WithCerts(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile := writeSelfSignedCerts(t, dir)

	creds, err := tlsCreds(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Info().SecurityProtocol != "tls" {
		t.Errorf("expected tls protocol, got %q", creds.Info().SecurityProtocol)
	}
}

func TestTLSCreds_MissingCA(t *testing.T) {
	// The CA cert is read eagerly at creds creation time; a missing CA must
	// return an error immediately (cert/key are loaded lazily at handshake).
	dir := t.TempDir()
	certFile, keyFile, _ := writeSelfSignedCerts(t, dir)
	_, err := tlsCreds(certFile, keyFile, "/nonexistent/ca.pem")
	if err == nil {
		t.Error("expected error for missing CA cert, got nil")
	}
}

// writeSelfSignedCerts generates a self-signed CA + leaf cert and writes PEM
// files to dir. Returns (certFile, keyFile, caFile) paths.
func writeSelfSignedCerts(t *testing.T, dir string) (string, string, string) {
	t.Helper()

	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}

	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-service"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
	}
	caCert, _ := x509.ParseCertificate(caDER)
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}

	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	caFile := filepath.Join(dir, "ca.pem")

	writePEM(t, certFile, "CERTIFICATE", leafDER)
	leafKeyDER, _ := x509.MarshalECPrivateKey(leafKey)
	writePEM(t, keyFile, "EC PRIVATE KEY", leafKeyDER)
	writePEM(t, caFile, "CERTIFICATE", caDER)

	return certFile, keyFile, caFile
}

func writePEM(t *testing.T, path, pemType string, der []byte) {
	t.Helper()
	f, err := os.Create(path) //nolint:gosec
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if err := pem.Encode(f, &pem.Block{Type: pemType, Bytes: der}); err != nil {
		_ = f.Close()
		t.Fatalf("encode PEM %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}
