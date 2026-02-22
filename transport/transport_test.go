package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeTLSReturnsListenErrorImmediately(t *testing.T) {
	certFile, keyFile := writeSelfSignedCertFiles(t)

	blocked, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建占位监听失败: %v", err)
	}
	defer blocked.Close()

	_, err = ServeTLS(context.Background(), Config{
		Addrs:    []string{blocked.Addr().String()},
		CertFile: certFile,
		KeyFile:  keyFile,
		ConnService: func(ctx context.Context, conn net.Conn) error {
			return nil
		},
	})
	if err == nil {
		t.Fatalf("期望监听失败时返回错误，但得到 nil")
	}
	if !strings.Contains(err.Error(), "监听失败") {
		t.Fatalf("期望错误包含监听失败，实际: %v", err)
	}
}

func writeSelfSignedCertFiles(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("生成证书失败: %v", err)
	}

	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "server.crt")
	keyFile := filepath.Join(tempDir, "server.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatalf("写入证书失败: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}

	return certFile, keyFile
}
