package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"testing"
)

func TestGenerateECP256Key(t *testing.T) {
	key, err := GenerateECP256Key()
	if err != nil {
		t.Fatalf("GenerateECP256Key: %v", err)
	}
	if key.Curve != elliptic.P256() {
		t.Errorf("got curve %v, want P-256", key.Curve.Params().Name)
	}
}

func TestCreateCSRDER_RoundTrip(t *testing.T) {
	key, err := GenerateECP256Key()
	if err != nil {
		t.Fatalf("GenerateECP256Key: %v", err)
	}

	const cn = "some/path/to/AC-01"
	der, err := CreateCSRDER(key, cn, "ac-01.example.com")
	if err != nil {
		t.Fatalf("CreateCSRDER: %v", err)
	}

	csr, err := ParseCSRDER(der)
	if err != nil {
		t.Fatalf("ParseCSRDER: %v", err)
	}

	if csr.Subject.CommonName != cn {
		t.Errorf("CommonName = %q, want %q", csr.Subject.CommonName, cn)
	}
	if len(csr.DNSNames) != 1 || csr.DNSNames[0] != "ac-01.example.com" {
		t.Errorf("DNSNames = %v, want [ac-01.example.com]", csr.DNSNames)
	}
	pub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("CSR public key type = %T, want *ecdsa.PublicKey", csr.PublicKey)
	}
	if !pub.Equal(&key.PublicKey) {
		t.Error("CSR public key does not match the generated key")
	}
}

func TestParseCSRDER_RejectsGarbage(t *testing.T) {
	if _, err := ParseCSRDER([]byte("not der")); err == nil {
		t.Error("expected error for non-DER input")
	}
}
