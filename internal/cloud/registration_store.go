package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/renameio/v2/maybe"
)

// RegistrationStore persists and retrieves a Registration.
type RegistrationStore interface {
	Load(ctx context.Context) (Registration, bool, error)
	Save(ctx context.Context, reg Registration) error
	Clear(ctx context.Context) error
}

// NewFileRegistrationStore returns a RegistrationStore backed by a JSON file at path.
// Writes are atomic (temp-file + rename) and the file is created with 0600 permissions.
func NewFileRegistrationStore(path string) RegistrationStore {
	return &fileRegistrationStore{path: path}
}

type fileRegistrationStore struct {
	path string
}

func (s *fileRegistrationStore) Load(_ context.Context) (Registration, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Registration{}, false, nil
	}
	if err != nil {
		return Registration{}, false, fmt.Errorf("read registration file: %w", err)
	}
	var reg Registration
	if err := json.Unmarshal(data, &reg); err != nil {
		return Registration{}, false, fmt.Errorf("decode registration file: %w", err)
	}
	return reg, true, nil
}

func (s *fileRegistrationStore) Save(_ context.Context, reg Registration) error {
	data, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("encode registration: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create registration dir: %w", err)
	}
	return maybe.WriteFile(s.path, data, 0600)
}

func (s *fileRegistrationStore) Clear(_ context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
