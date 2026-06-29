package cloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// CredentialStore persists and retrieves the controller's cloud Credential
// (EC private key + Connect-CA certificate chain).
type CredentialStore interface {
	Load(ctx context.Context) (*Credential, bool, error)
	Save(ctx context.Context, cred *Credential) error
	Clear(ctx context.Context) error
}

// NewFileCredentialStore returns a CredentialStore backed by two PEM files: the
// private key at keyPath (written 0600) and the certificate chain at certPath
// (leaf first, written 0644). Writes are atomic (temp-file + rename). logger, if
// non-nil, is used to warn about a half-written credential (one file present
// without the other); pass zap.NewNop() to silence.
func NewFileCredentialStore(keyPath, certPath string, logger *zap.Logger) CredentialStore {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &fileCredentialStore{keyPath: keyPath, certPath: certPath, logger: logger}
}

type fileCredentialStore struct {
	keyPath  string
	certPath string
	logger   *zap.Logger
}

func (s *fileCredentialStore) Load(_ context.Context) (*Credential, bool, error) {
	key, _, err := pki.LoadPrivateKey(s.keyPath)
	if errors.Is(err, os.ErrNotExist) {
		// No key. A lone cert with no key is a half-written/partially-cleared
		// credential, not a clean empty store — boot unconfigured (so local
		// function is unaffected) but warn so the orphaned file is not silently
		// ignored and the operator knows to re-enroll.
		if _, statErr := os.Stat(s.certPath); statErr == nil {
			s.logger.Warn("ignoring half-written cloud credential: cert present but key missing; re-enrollment required",
				zap.String("certPath", s.certPath), zap.String("keyPath", s.keyPath))
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("load cloud key: %w", err)
	}

	certPEM, err := os.ReadFile(s.certPath)
	if errors.Is(err, os.ErrNotExist) {
		// Key present but cert missing — a half-written credential leaving an
		// orphaned private key. Boot unconfigured but warn rather than silently
		// treating it as a clean empty store.
		s.logger.Warn("ignoring half-written cloud credential: key present but cert missing; re-enrollment required",
			zap.String("keyPath", s.keyPath), zap.String("certPath", s.certPath))
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read cloud cert: %w", err)
	}
	chain, err := pki.ParseCertificatesPEM(certPEM)
	if err != nil {
		return nil, false, fmt.Errorf("parse cloud cert chain: %w", err)
	}

	cred, err := newCredential(key, chain)
	if err != nil {
		return nil, false, fmt.Errorf("load cloud credential: %w", err)
	}
	return cred, true, nil
}

func (s *fileCredentialStore) Save(_ context.Context, cred *Credential) error {
	keyPEM, err := pki.EncodePrivateKey(cred.Key)
	if err != nil {
		return fmt.Errorf("encode cloud key: %w", err)
	}
	certPEM := pki.EncodeCertificates(cred.Chain)

	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0700); err != nil {
		return fmt.Errorf("create cloud credential dir: %w", err)
	}
	if err := atomicWrite(s.keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("write cloud key: %w", err)
	}
	if err := atomicWrite(s.certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("write cloud cert: %w", err)
	}
	return nil
}

func (s *fileCredentialStore) Clear(_ context.Context) error {
	var errs error
	for _, p := range []string{s.keyPath, s.certPath} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// atomicWrite writes data to path via a temp file in the same directory followed
// by a rename, so a reader never observes a partially-written file. The file is
// created with perm.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cred-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		cleanup()
		return fmt.Errorf("set permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
