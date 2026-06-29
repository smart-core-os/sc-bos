package cloud

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// testCredential builds a self-signed credential whose leaf CN is nodeID.
func testCredential(t *testing.T, nodeID string) *Credential {
	t.Helper()
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("GenerateECP256Key: %v", err)
	}
	der, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject: pkix.Name{CommonName: nodeID},
	}, key)
	if err != nil {
		t.Fatalf("CreateSelfSignedCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	cred, err := newCredential(key, []*x509.Certificate{leaf})
	if err != nil {
		t.Fatalf("newCredential: %v", err)
	}
	return cred
}

func TestCredential_NodeIDAndTLS(t *testing.T) {
	cred := testCredential(t, "node-123")
	if got := cred.NodeID(); got != "node-123" {
		t.Errorf("NodeID = %q, want node-123", got)
	}
	tlsCert := cred.TLSCertificate()
	if len(tlsCert.Certificate) != 1 {
		t.Fatalf("tls cert has %d certs, want 1", len(tlsCert.Certificate))
	}
	if tlsCert.PrivateKey != cred.Key {
		t.Error("tls cert private key is not the credential key")
	}
}

func TestFileCredentialStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileCredentialStore(
		filepath.Join(dir, "cloud.key.pem"),
		filepath.Join(dir, "cloud.cert.pem"),
		zap.NewNop(),
	)
	ctx := context.Background()

	// nothing stored yet
	if _, ok, err := store.Load(ctx); err != nil || ok {
		t.Fatalf("Load on empty store: ok=%v err=%v", ok, err)
	}

	cred := testCredential(t, "node-abc")
	if err := store.Save(ctx, cred); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, ok, err := store.Load(ctx)
	if err != nil || !ok {
		t.Fatalf("Load after save: ok=%v err=%v", ok, err)
	}
	if loaded.NodeID() != "node-abc" {
		t.Errorf("loaded NodeID = %q, want node-abc", loaded.NodeID())
	}
	if !loaded.Leaf().Equal(cred.Leaf()) {
		t.Error("loaded leaf differs from saved leaf")
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, ok, err := store.Load(ctx); err != nil || ok {
		t.Fatalf("Load after clear: ok=%v err=%v", ok, err)
	}
}
