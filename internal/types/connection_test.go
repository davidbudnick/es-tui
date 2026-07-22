package types

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConnectionAddressAndBaseURL(t *testing.T) {
	c := Connection{Host: "localhost", Port: 9200}
	if c.Address() != "localhost:9200" {
		t.Fatalf("Address: got %s", c.Address())
	}
	if c.BaseURL() != "http://localhost:9200" {
		t.Fatalf("BaseURL: got %s", c.BaseURL())
	}
	c.UseTLS = true
	if c.BaseURL() != "https://localhost:9200" {
		t.Fatalf("BaseURL TLS: got %s", c.BaseURL())
	}
}

func TestTLSConfigBuildEmpty(t *testing.T) {
	cfg := &TLSConfig{InsecureSkipVerify: true, ServerName: "example.com"}
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	if !tlsCfg.InsecureSkipVerify || tlsCfg.ServerName != "example.com" {
		t.Fatalf("unexpected tls config: %+v", tlsCfg)
	}
}

func TestTLSConfigBuildWithCerts(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile := writeTestCerts(t, dir)

	cfg := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(tlsCfg.Certificates))
	}
	if tlsCfg.RootCAs == nil {
		t.Fatal("expected RootCAs")
	}
}

func TestTLSConfigBuildBadKeyPair(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "bad.crt")
	keyFile := filepath.Join(dir, "bad.key")
	if err := os.WriteFile(certFile, []byte("not a cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &TLSConfig{CertFile: certFile, KeyFile: keyFile}
	if _, err := cfg.BuildTLSConfig(); err == nil {
		t.Fatal("expected error for bad key pair")
	}
}

func TestTLSConfigBuildMissingCA(t *testing.T) {
	cfg := &TLSConfig{CAFile: filepath.Join(t.TempDir(), "missing.pem")}
	if _, err := cfg.BuildTLSConfig(); err == nil {
		t.Fatal("expected error for missing CA")
	}
}

func TestTLSConfigBuildBadCA(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caFile, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &TLSConfig{CAFile: caFile}
	if _, err := cfg.BuildTLSConfig(); err == nil {
		t.Fatal("expected error for bad CA")
	}
}

func writeTestCerts(t *testing.T, dir string) (certFile, keyFile, caFile string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	caFile = filepath.Join(dir, "ca.pem")

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatal(err)
	}
	if err := certOut.Close(); err != nil {
		t.Fatal(err)
	}

	// CA same as cert for test
	if err := os.WriteFile(caFile, mustRead(t, certFile), 0o600); err != nil {
		t.Fatal(err)
	}

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatal(err)
	}
	if err := keyOut.Close(); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile, caFile
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
