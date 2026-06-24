// Package selfupdate implements the Supervisor updating itself from an RPM.
//
// Because the Supervisor is the process being replaced, it cannot watch its own replacement: the work
// runs in a separate, short-lived applier process (a transient systemd unit, so it outlives the
// restart the RPM triggers). The in-process Updater accepts an InstallSupervisorUpdate request, records
// the intent, and launches the applier; the Applier downloads + verifies the RPM, installs it, confirms
// the new version comes up healthy, and rolls back to the last-known-good RPM if it does not.
package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/google/renameio/v2/maybe"
)

// Phase is the lifecycle of a self-update attempt, persisted so the applier (a separate process) and
// the restarted Supervisor share one view of it.
type Phase string

const (
	// PhaseInstalling: the intent is recorded; the applier is downloading/installing the new RPM.
	PhaseInstalling Phase = "installing"
	// PhaseRollingBack: the new version did not come up healthy; the applier is reinstalling the previous.
	PhaseRollingBack Phase = "rolling_back"
	// PhaseCompleted: the new version is installed and confirmed healthy.
	PhaseCompleted Phase = "completed"
	// PhaseFailed: the update failed; the Supervisor was rolled back to the previous version (or never
	// left it).
	PhaseFailed Phase = "failed"
)

// Terminal reports whether the phase is a settled outcome (no applier is still working).
func (p Phase) Terminal() bool { return p == PhaseCompleted || p == PhaseFailed }

// State is the durable record of the most recent self-update. The Updater writes the intent
// (Target/URL/SHA256/Previous*) before launching the applier; the applier advances Phase and writes the
// outcome. It is read by GetSupervisorInfo so BOS can relay the result.
type State struct {
	Target          string     `json:"target"`                    // the version being installed
	URL             string     `json:"url"`                       // capability URL to download the RPM
	SHA256          string     `json:"sha256"`                    // hex-encoded expected digest
	Previous        string     `json:"previous,omitempty"`        // last-known-good version (rollback target)
	PreviousRPMPath string     `json:"previousRpmPath,omitempty"` // path to the rollback RPM ("" if none kept)
	Phase           Phase      `json:"phase"`
	Error           string     `json:"error,omitempty"`
	StartTime       time.Time  `json:"startTime"`
	FinishTime      *time.Time `json:"finishTime,omitempty"`
}

// Store persists a single State as JSON at Path. Writes are atomic (temp + rename).
type Store struct {
	Path string
}

// Load returns the persisted state. The boolean is false (with a nil error) when no state file exists.
func (s *Store) Load() (State, bool, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, fmt.Errorf("read self-update state: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, false, fmt.Errorf("decode self-update state: %w", err)
	}
	return st, true, nil
}

// Save atomically writes st.
func (s *Store) Save(st State) error {
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("encode self-update state: %w", err)
	}
	if err := maybe.WriteFile(s.Path, data, 0o600); err != nil {
		return fmt.Errorf("write self-update state: %w", err)
	}
	return nil
}
