package cloud

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/renameio/v2"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// RegistrationStore persists and retrieves the controller's cloud Registration
// (API endpoint + EC private key + Connect-CA certificate chain).
type RegistrationStore interface {
	Load(ctx context.Context) (*Registration, bool, error)
	Save(ctx context.Context, reg *Registration) error
	Clear(ctx context.Context) error
}

// NewFileRegistrationStore returns a RegistrationStore backed by a single JSON
// file at path (written atomically, 0600). Holding the key, certificate chain
// and API endpoint in one file means they are always mutually consistent — a
// crash can never leave a key without its certificate, or a certificate issued
// by one origin paired with a different endpoint.
func NewFileRegistrationStore(path string) RegistrationStore {
	return &fileRegistrationStore{path: path}
}

type fileRegistrationStore struct {
	path string
}

// registrationJSON is the on-disk representation: PEM for the key and chain so
// the file is human-inspectable, plus the API endpoint.
type registrationJSON struct {
	APIEndpoint string `json:"apiEndpoint"`
	Key         string `json:"key"`         // PEM (PKCS#8)
	Certificate string `json:"certificate"` // PEM chain, leaf first
}

func (s *fileRegistrationStore) Load(_ context.Context) (*Registration, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read registration: %w", err)
	}

	var dto registrationJSON
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, false, fmt.Errorf("decode registration: %w", err)
	}

	block, _ := pem.Decode([]byte(dto.Key))
	if block == nil {
		return nil, false, fmt.Errorf("registration: invalid key PEM")
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, false, fmt.Errorf("registration: parse key: %w", err)
	}
	key, ok := keyAny.(pki.PrivateKey)
	if !ok {
		return nil, false, fmt.Errorf("registration: unexpected key type %T", keyAny)
	}
	chain, err := pki.ParseCertificatesPEM([]byte(dto.Certificate))
	if err != nil {
		return nil, false, fmt.Errorf("registration: parse certificate chain: %w", err)
	}

	reg, err := newRegistration(key, chain, dto.APIEndpoint)
	if err != nil {
		return nil, false, fmt.Errorf("registration: %w", err)
	}
	return reg, true, nil
}

func (s *fileRegistrationStore) Save(_ context.Context, reg *Registration) error {
	keyPEM, err := pki.EncodePrivateKey(reg.Key)
	if err != nil {
		return fmt.Errorf("encode key: %w", err)
	}
	data, err := json.Marshal(registrationJSON{
		APIEndpoint: reg.APIEndpoint,
		Key:         string(keyPEM),
		Certificate: string(pki.EncodeCertificates(reg.Chain)),
	})
	if err != nil {
		return fmt.Errorf("encode registration: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create registration dir: %w", err)
	}
	// renameio writes to a temp file in the same directory and renames it into
	// place, so a reader never sees a partial file and a crash never corrupts
	// the existing registration.
	if err := renameio.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write registration: %w", err)
	}
	return nil
}

func (s *fileRegistrationStore) Clear(_ context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
