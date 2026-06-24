package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/renameio/v2/maybe"
)

// UpdateState is the persisted record of an in-flight software update: BOS has accepted an update
// deployment from the cloud and intends to (or is in the middle of) installing it. Absence of the
// record means there is no in-flight update.
type UpdateState struct {
	DeploymentID string    `json:"deploymentId"`
	Version      string    `json:"version"`
	Attempts     int       `json:"attempts,omitempty"`
	StartTime    time.Time `json:"startTime"`
}

// UpdateStore persists and retrieves the in-flight UpdateState.
type UpdateStore interface {
	// Load returns the persisted UpdateState. ok is false (and the returned UpdateState is the zero
	// value) when there is no in-flight update on disk.
	Load(ctx context.Context) (state UpdateState, ok bool, err error)
	// Save atomically persists state.
	Save(ctx context.Context, state UpdateState) error
	// Clear removes any persisted state. Clearing an absent record is not an error.
	Clear(ctx context.Context) error
}

// NewFileUpdateStore returns an UpdateStore backed by a JSON file at path.
// Writes are atomic (temp-file + rename) and the file is created with 0600 permissions.
func NewFileUpdateStore(path string) UpdateStore {
	return &fileUpdateStore{path: path}
}

type fileUpdateStore struct {
	path string
}

func (s *fileUpdateStore) Load(_ context.Context) (UpdateState, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return UpdateState{}, false, nil
	}
	if err != nil {
		return UpdateState{}, false, fmt.Errorf("read update file: %w", err)
	}
	var state UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return UpdateState{}, false, fmt.Errorf("decode update file: %w", err)
	}
	return state, true, nil
}

func (s *fileUpdateStore) Save(_ context.Context, state UpdateState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode update state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create update dir: %w", err)
	}
	return maybe.WriteFile(s.path, data, 0600)
}

func (s *fileUpdateStore) Clear(_ context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
