package cloud

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// Credential is the controller's cloud identity under mTLS: an EC P-256 private
// key together with the Connect-CA-issued certificate chain (leaf first).
//
// The API endpoint base is deliberately not held here — the contract returns no
// API root, so it is derived from the configured register URL.
type Credential struct {
	Key   pki.PrivateKey
	Chain []*x509.Certificate // leaf first, followed by any intermediates/CA
}

// Leaf returns the controller's own (leaf) certificate.
func (c *Credential) Leaf() *x509.Certificate {
	return c.Chain[0]
}

// NodeID returns the SCC node id, which the CA sets as the leaf certificate's
// subject common name. It is stable across renewals.
func (c *Credential) NodeID() string {
	return c.Chain[0].Subject.CommonName
}

// TLSCertificate adapts the credential into a *tls.Certificate suitable for
// presentation as a client certificate.
func (c *Credential) TLSCertificate() *tls.Certificate {
	cert := &tls.Certificate{
		PrivateKey:  c.Key,
		Leaf:        c.Chain[0],
		Certificate: make([][]byte, 0, len(c.Chain)),
	}
	for _, x := range c.Chain {
		cert.Certificate = append(cert.Certificate, x.Raw)
	}
	return cert
}

// newCredential validates that chain is non-empty, leaf-first, and that its leaf
// public key matches key, returning a Credential ready for use.
func newCredential(key pki.PrivateKey, chain []*x509.Certificate) (*Credential, error) {
	if len(chain) == 0 {
		return nil, fmt.Errorf("certificate chain is empty")
	}
	if err := pki.ValidKeyPair(chain[0].PublicKey, key); err != nil {
		return nil, fmt.Errorf("leaf certificate does not match private key: %w", err)
	}
	return &Credential{Key: key, Chain: chain}, nil
}
