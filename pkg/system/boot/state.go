package boot

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RebootState is the on-disk format for persisting the last reboot reason/actor across restarts.
type RebootState struct {
	Reason    string          `json:"reason,omitempty"`
	Actor     json.RawMessage `json:"actor,omitempty"`
	CleanExit bool            `json:"cleanExit,omitempty"`
}

// WriteStateFile atomically writes st to the reboot-state.json file in dataDir.
// If dataDir is empty it is a no-op.
func WriteStateFile(dataDir string, st RebootState) error {
	if dataDir == "" {
		return nil
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	dst := filepath.Join(dataDir, "reboot-state.json")
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// ReadStateFile reads and deserialises the reboot-state.json file from dataDir.
func ReadStateFile(dataDir string) (RebootState, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "reboot-state.json"))
	if err != nil {
		return RebootState{}, err
	}
	var st RebootState
	if err := json.Unmarshal(data, &st); err != nil {
		return RebootState{}, err
	}
	return st, nil
}
