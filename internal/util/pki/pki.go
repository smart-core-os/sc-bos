package pki

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"go.uber.org/multierr"
)

// SaveCertificateChain writes derCerts as pem encoded CERTIFICATE blocks to certFile.
// Returns the written bytes.
func SaveCertificateChain(certFile string, derCerts [][]byte) (pemBytes []byte, err error) {
	var buf bytes.Buffer

	for _, derCert := range derCerts {
		err = pem.Encode(&buf, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derCert,
		})
		if err != nil {
			return
		}
	}

	pemBytes = buf.Bytes()
	err = writeFileNoTruncate(certFile, pemBytes, 0640)
	return
}

// ParseCertificatesPEM parses and returns any CERTIFICATE blocks found in the pem encoded pemBytes.
func ParseCertificatesPEM(pemBytes []byte) (certs []*x509.Certificate, errs error) {
	for len(pemBytes) > 0 {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			return
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}

		certs = append(certs, cert)
	}
	return
}

// DecodePEMBlocks returns all PEM blocks in pemBytes with blockType type.
func DecodePEMBlocks(pemBytes []byte, blockType string) [][]byte {
	var matched [][]byte
	for len(pemBytes) > 0 {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if block.Type != blockType {
			continue
		}

		matched = append(matched, block.Bytes)
	}
	return matched
}

func EncodePEMSequence(contents [][]byte, blockType string) (encoded []byte) {
	var buf bytes.Buffer

	for _, blockData := range contents {
		block := &pem.Block{
			Type:  blockType,
			Bytes: blockData,
		}

		err := pem.Encode(&buf, block)
		if err != nil {
			// buf.Write never returns an error, so pem.Encode won't either
			panic(err)
		}
	}

	return buf.Bytes()
}

func EncodeCertificates(certs []*x509.Certificate) []byte {
	out := &bytes.Buffer{}
	block := &pem.Block{Type: "CERTIFICATE"}
	for _, cert := range certs {
		if len(cert.Raw) == 0 {
			continue
		}
		block.Bytes = cert.Raw
		// writing to a memory buffer shouldn't return any errors
		err := pem.Encode(out, block)
		if err != nil {
			panic(err)
		}
	}
	return out.Bytes()
}

// GenerateSerialNumber generates a random unsigned 128-bit integer using a cryptographically secure source of random
// numbers.
// The returned integer is suitable for use as an X.509 certificate serial number.
func GenerateSerialNumber() (*big.Int, error) {
	blob := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, blob)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(blob), nil
}

// LoadX509Cert reads an x509 certificate chain (leaf first) from certPath and combines it with privateKey to form a tls.Certificate.
// This is like tls.LoadX509KeyPair except the private key is already known.
// Leaf will be populated with the first certificate in certPath.
func LoadX509Cert(certPath string, privateKey crypto.PrivateKey) (tls.Certificate, error) {
	fail := func(err error) (tls.Certificate, error) {
		return tls.Certificate{}, err
	}

	certPem, err := os.ReadFile(certPath)
	if err != nil {
		return fail(err)
	}
	cert := tls.Certificate{}
	cert.Certificate = DecodePEMBlocks(certPem, "CERTIFICATE")
	cert.PrivateKey = privateKey

	if len(cert.Certificate) == 0 {
		return fail(fmt.Errorf("no certificates in %v", certPem))
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fail(err)
	}
	if err := ValidKeyPair(cert.Leaf.PublicKey, privateKey); err != nil {
		return fail(err)
	}

	return cert, nil
}

// ValidKeyPair checks whether the given public and private keys are a valid pair, that is to say they use the same
// algorithm configured with the same parameters.
// If the keys are not a valid pair a non-nil error will be returned.
func ValidKeyPair(public crypto.PublicKey, private crypto.PrivateKey) error {
	// Code copied from tls.X509KeyPair
	switch pub := public.(type) {
	case *rsa.PublicKey:
		priv, ok := private.(*rsa.PrivateKey)
		if !ok {
			return errors.New("tls: private key type does not match public key type")
		}
		if !priv.PublicKey.Equal(pub) {
			return errors.New("tls: private key does not match public key")
		}
	case *ecdsa.PublicKey:
		priv, ok := private.(*ecdsa.PrivateKey)
		if !ok {
			return errors.New("tls: private key type does not match public key type")
		}
		if !priv.PublicKey.Equal(pub) {
			return errors.New("tls: private key does not match public key")
		}
	case ed25519.PublicKey:
		priv, ok := private.(ed25519.PrivateKey)
		if !ok {
			return errors.New("tls: private key type does not match public key type")
		}
		if !priv.Public().(ed25519.PublicKey).Equal(pub) {
			return errors.New("tls: private key does not match public key")
		}
	default:
		return errors.New("tls: unknown public key algorithm")
	}

	return nil
}

// LoadCertAndRootsWithKey reads and returns a certificate from certPath and validates it against key.
// It optionally reads a roots file from rootsPath, if rootsPath is empty or the file is not found then returns nil.
func LoadCertAndRootsWithKey(certPath, rootsPath string, key PrivateKey) (*tls.Certificate, []*x509.Certificate, error) {
	cert, err := LoadX509Cert(certPath, key)
	if err != nil {
		return nil, nil, err
	}
	if rootsPath == "" {
		return &cert, nil, nil
	}

	rootsPem, err := os.ReadFile(rootsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// we ignore that roots doesn't exist, this just means we don't trust other nodes
			return &cert, nil, nil
		}
		return nil, nil, err
	}
	roots, err := ParseCertificatesPEM(rootsPem)
	if err != nil {
		return nil, nil, err
	}
	return &cert, roots, nil
}
