package cloud

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// TestCredential_IntermediateChain verifies the controller works when SCC issues
// its certificate through an intermediate CA (root -> intermediate -> leaf): the
// full chain is stored, and the intermediate is presented on the mTLS handshake
// so a server that trusts only the root can still build the path.
func TestCredential_IntermediateChain(t *testing.T) {
	leafKey, chain, rootCert := buildIntermediateChain(t, "node-abc")
	if len(chain) != 3 {
		t.Fatalf("expected a 3-cert chain [leaf, intermediate, root], got %d", len(chain))
	}

	cred, err := newRegistration(leafKey, chain, "")
	if err != nil {
		t.Fatalf("newRegistration: %v", err)
	}
	if cred.NodeID() != "node-abc" {
		t.Errorf("NodeID = %q, want node-abc (leaf CN)", cred.NodeID())
	}
	if got := len(cred.TLSCertificate().Certificate); got != 3 {
		t.Errorf("TLSCertificate presents %d certs, want 3 (leaf + intermediate + root)", got)
	}

	// The full chain must survive a store round-trip (so it persists across restart).
	regStore, _ := newStores(t)
	if err := regStore.Save(context.Background(), cred); err != nil {
		t.Fatalf("save: %v", err)
	}
	reloaded, ok, err := regStore.Load(context.Background())
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if len(reloaded.Chain) != 3 {
		t.Errorf("reloaded chain has %d certs, want 3", len(reloaded.Chain))
	}

	// An mTLS server that trusts ONLY the root and requires a verified client
	// certificate. Verification succeeds only if the controller presents the
	// intermediate so the server can build leaf -> intermediate -> root.
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"checkIn": map[string]any{"nodeId": "node-abc", "checkInTime": "2026-01-01T00:00:00Z"},
		})
	}))
	ts.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: rootPool}
	ts.StartTLS()
	defer ts.Close()

	t.Run("intermediate presented: check-in succeeds", func(t *testing.T) {
		client := NewHTTPClient(cred, ts.URL, WithInsecureSkipVerify())
		if _, err := client.CheckIn(context.Background(), CheckInRequest{}); err != nil {
			t.Fatalf("check-in with an intermediate chain failed: %v", err)
		}
	})

	t.Run("intermediate omitted: check-in fails", func(t *testing.T) {
		// Same leaf and key, but the stored chain skips the intermediate. The
		// server cannot build a path from the leaf to the root it trusts.
		partial, err := newRegistration(leafKey, []*x509.Certificate{chain[0], rootCert}, "")
		if err != nil {
			t.Fatalf("newRegistration: %v", err)
		}
		client := NewHTTPClient(partial, ts.URL, WithInsecureSkipVerify())
		if _, err := client.CheckIn(context.Background(), CheckInRequest{}); err == nil {
			t.Error("expected check-in to fail when the intermediate is not presented")
		}
	})
}

// buildIntermediateChain builds root -> intermediate -> leaf and returns the leaf
// private key, the leaf-first chain [leaf, intermediate, root], and the root
// certificate (for a server's ClientCAs pool).
func buildIntermediateChain(t *testing.T, leafCN string) (pki.PrivateKey, []*x509.Certificate, *x509.Certificate) {
	t.Helper()

	rootKey, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("root key: %v", err)
	}
	rootDER, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}, rootKey)
	if err != nil {
		t.Fatalf("root cert: %v", err)
	}
	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root: %v", err)
	}
	rootTLS := &tls.Certificate{Certificate: [][]byte{rootDER}, PrivateKey: rootKey, Leaf: rootCert}

	interKey, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("intermediate key: %v", err)
	}
	interPEM, err := pki.CreateCertificateChain(rootTLS, &x509.Certificate{
		Subject:               pkix.Name{CommonName: "Test Intermediate CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}, interKey.Public())
	if err != nil {
		t.Fatalf("intermediate cert: %v", err)
	}
	interChain, err := pki.ParseCertificatesPEM(interPEM)
	if err != nil || len(interChain) == 0 {
		t.Fatalf("parse intermediate: %v", err)
	}
	interCert := interChain[0]
	interTLS := &tls.Certificate{Certificate: [][]byte{interCert.Raw, rootDER}, PrivateKey: interKey, Leaf: interCert}

	leafKey, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafPEM, err := pki.CreateCertificateChain(interTLS, &x509.Certificate{
		Subject:     pkix.Name{CommonName: leafCN},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, leafKey.Public())
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}
	chain, err := pki.ParseCertificatesPEM(leafPEM)
	if err != nil {
		t.Fatalf("parse leaf chain: %v", err)
	}
	return leafKey, chain, rootCert
}
