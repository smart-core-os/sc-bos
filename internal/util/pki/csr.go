package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
)

// GenerateECP256Key generates a new NIST P-256 (secp256r1) ECDSA private key.
//
// This is the key type used for SCC device certificates: the Connect CA rejects
// a CSR carrying any other key type.
func GenerateECP256Key() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// CreateCSRDER builds a DER-encoded PKCS#10 certificate signing request for key,
// carrying commonName in the subject. Any dnsSANs are added as DNS subject
// alternative names.
//
// The result is raw DER (the wire format for application/pkcs10); callers
// transferring it over HTTP base64-encode it themselves
// (Content-Transfer-Encoding: base64). The signature algorithm is derived from
// key (ECDSA P-256 keys sign with ECDSA-SHA256).
func CreateCSRDER(key crypto.Signer, commonName string, dnsSANs ...string) ([]byte, error) {
	tmpl := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: commonName},
		DNSNames: dnsSANs,
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tmpl, key)
	if err != nil {
		return nil, fmt.Errorf("create certificate request: %w", err)
	}
	return der, nil
}

// ParseCSRDER parses a DER-encoded PKCS#10 certificate signing request and
// verifies its signature. It is the inverse of CreateCSRDER and is used by the
// signing side (e.g. the cloud simulator) to read a submitted request.
func ParseCSRDER(der []byte) (*x509.CertificateRequest, error) {
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return nil, fmt.Errorf("parse certificate request: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("verify certificate request signature: %w", err)
	}
	return csr, nil
}
