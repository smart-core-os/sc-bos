package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create registration dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".reg-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write registration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("set permissions: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
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
