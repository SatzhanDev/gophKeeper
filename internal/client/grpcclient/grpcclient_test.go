package grpcclient

import (
	"context"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// writeTestCert генерирует минимальный самоподписанный сертификат во
// временный файл — тесту нужен только валидный PEM, который сможет
// распарсить Dial, реальное TLS-соединение здесь не устанавливается
// (поэтому не нужно ни поднимать сервер, ни выполнять `make certs`).
func writeTestCert(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "test-cert.pem")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return path
}

func TestDial(t *testing.T) {
	certFile := writeTestCert(t)

	// grpc.NewClient не устанавливает соединение немедленно (ленивое
	// подключение), поэтому Dial успешно отработает даже без реального
	// сервера на этом адресе — реальная попытка соединения произойдёт
	// только при первом RPC-вызове.
	conn, err := Dial("localhost:1", certFile)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()
}

func TestDial_InvalidCertFile(t *testing.T) {
	_, err := Dial("localhost:1", "/no/such/file.crt")
	assert.Error(t, err)
}

func TestAuthContext(t *testing.T) {
	ctx := AuthContext(context.Background(), "my-token")

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer my-token"}, md.Get("authorization"))
}
