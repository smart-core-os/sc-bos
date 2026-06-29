package sim

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// credentialURIPrefix namespaces the opaque credentialId carried in a device
// certificate's URI SAN. BOS treats the credentialId as opaque; only the sim
// (standing in for SCC) reads it back out. The prefix mirrors the real SCC CA
// (urn:smartcore:credential-id:) so a sim-issued cert has the same SAN shape.
const credentialURIPrefix = "urn:smartcore:credential-id:"

// Credential slot names, mirroring the contract's two interchangeable slots.
const (
	slotPrimary   = "primary"
	slotSecondary = "secondary"
)

// defaultCertLifetime is the issued device-certificate lifetime. Short relative
// to production (~1 year) so renewal can be exercised quickly in tests.
const defaultCertLifetime = 90 * 24 * time.Hour

// devCA is the simulator's stand-in for the Connect device CA. It signs device
// certificates and provides the trust anchor used to verify client certificates
// (and to present a server certificate for local mTLS).
type devCA struct {
	tlsCert *tls.Certificate // CA leaf + key, used as the signing authority
	cert    *x509.Certificate
	pool    *x509.CertPool // trust anchor for verifying client/server certs
}

// newDevCA generates a fresh EC P-256 CA key and a self-signed CA certificate.
func newDevCA() (*devCA, error) {
	key, err := pki.GenerateECP256Key()
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}
	tmpl := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "SCC Dev Connect CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := pki.CreateSelfSignedCertificate(tmpl, key, pki.WithExpireAfter(10*365*24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &devCA{
		tlsCert: &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert},
		cert:    cert,
		pool:    pool,
	}, nil
}

// signDevice issues a device leaf certificate for nodeID carrying credentialID
// in a URI SAN, per the device certificate profile (CN=nodeID, EKU clientAuth,
// KU digitalSignature, CA:FALSE). It returns the PEM chain (leaf first, then CA)
// and the parsed leaf for recording its metadata.
func (ca *devCA) signDevice(pub crypto.PublicKey, nodeID, credentialID string, lifetime time.Duration) (chainPEM []byte, leaf *x509.Certificate, err error) {
	credURI, err := url.Parse(credentialURIPrefix + credentialID)
	if err != nil {
		return nil, nil, fmt.Errorf("build credential URI: %w", err)
	}
	tmpl := &x509.Certificate{
		Subject:               pkix.Name{CommonName: nodeID},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{credURI},
	}
	chainPEM, err = pki.CreateCertificateChain(ca.tlsCert, tmpl, pub, pki.WithExpireAfter(lifetime))
	if err != nil {
		return nil, nil, fmt.Errorf("sign device certificate: %w", err)
	}
	leafDERs := pki.DecodePEMBlocks(chainPEM, "CERTIFICATE")
	if len(leafDERs) == 0 {
		return nil, nil, fmt.Errorf("signed chain has no certificates")
	}
	leaf, err = x509.ParseCertificate(leafDERs[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse signed leaf: %w", err)
	}
	return chainPEM, leaf, nil
}

// serverTLSConfig issues a server certificate (for localhost/127.0.0.1) from the
// CA and returns a TLS config that presents it and accepts a client certificate
// when offered (clientCertificateMode: accept — the per-endpoint guard does the
// real validation). Trust for both directions is the CA itself, which is what
// the local mTLS test harness uses in place of a public server cert.
func (ca *devCA) serverTLSConfig() (*tls.Config, error) {
	key, err := pki.GenerateECP256Key()
	if err != nil {
		return nil, fmt.Errorf("generate server key: %w", err)
	}
	tmpl := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	chainPEM, err := pki.CreateCertificateChain(ca.tlsCert, tmpl, key.Public(),
		pki.WithExpireAfter(10*365*24*time.Hour), pki.WithIfaces())
	if err != nil {
		return nil, fmt.Errorf("sign server certificate: %w", err)
	}
	serverCert := &tls.Certificate{
		Certificate: pki.DecodePEMBlocks(chainPEM, "CERTIFICATE"),
		PrivateKey:  key,
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*serverCert},
		ClientCAs:    ca.pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}, nil
}

// mintCredentialID returns a fresh opaque credential marker. It is unique by
// construction (random) and treated as opaque by BOS.
func mintCredentialID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "cred_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

// credentialIDFromCert extracts the credentialId from a device leaf's URI SANs,
// returning "" if absent.
func credentialIDFromCert(cert *x509.Certificate) string {
	for _, u := range cert.URIs {
		if s := u.String(); strings.HasPrefix(s, credentialURIPrefix) {
			return strings.TrimPrefix(s, credentialURIPrefix)
		}
	}
	return ""
}

// leafFingerprint returns the hex-encoded SHA-256 of a certificate's DER, used
// as a stable per-certificate identifier in the credential store.
func leafFingerprint(cert *x509.Certificate) []byte {
	sum := sha256.Sum256(cert.Raw)
	return sum[:]
}

// serialHex formats a certificate serial number as a hex string for storage.
func serialHex(cert *x509.Certificate) string {
	return hex.EncodeToString(cert.SerialNumber.Bytes())
}
