package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteControllerRebootState_nilErr(t *testing.T) {
	dir := t.TempDir()
	writeControllerRebootState(dir, nil)

	st := readRebootStateFile(t, dir)
	if !st.CleanExit {
		t.Errorf("CleanExit = false, want true")
	}
	if st.Reason != "" {
		t.Errorf("Reason = %q, want empty", st.Reason)
	}
}

func TestWriteControllerRebootState_deploymentRestart(t *testing.T) {
	dir := t.TempDir()
	writeControllerRebootState(dir, restartNowError{reason: "apply new deployment"})

	st := readRebootStateFile(t, dir)
	if !st.CleanExit {
		t.Errorf("CleanExit = false, want true")
	}
	if st.Reason != "apply new deployment" {
		t.Errorf("Reason = %q, want %q", st.Reason, "apply new deployment")
	}
}

func TestWriteControllerRebootState_bootRPCReboot(t *testing.T) {
	// restartNowError with empty reason signals that the boot system already wrote the state
	// file (with actor info); the controller must not overwrite it.
	dir := t.TempDir()
	prior := []byte(`{"reason":"operator-requested","cleanExit":true}`)
	_ = os.WriteFile(filepath.Join(dir, "reboot-state.json"), prior, 0o644)

	writeControllerRebootState(dir, restartNowError{reason: ""})

	got, _ := os.ReadFile(filepath.Join(dir, "reboot-state.json"))
	if string(got) != string(prior) {
		t.Errorf("state file overwritten:\n got  %s\n want %s", got, prior)
	}
}

func TestWriteControllerRebootState_unexpectedErr(t *testing.T) {
	// Any non-restart error leaves the in-progress marker in place (crash indicator).
	dir := t.TempDir()
	writeControllerRebootState(dir, errors.New("some unexpected error"))

	if _, err := os.ReadFile(filepath.Join(dir, "reboot-state.json")); !errors.Is(err, os.ErrNotExist) {
		t.Error("state file should not be written for unexpected errors")
	}
}

func TestWriteControllerRebootState_emptyDataDir(t *testing.T) {
	// Empty dataDir is a no-op; must not panic.
	writeControllerRebootState("", nil)
}

// readRebootStateFile reads and deserialises the state file written by writeControllerRebootState.
func readRebootStateFile(t *testing.T, dir string) controllerRebootState {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "reboot-state.json"))
	if err != nil {
		t.Fatalf("state file not written: %v", err)
	}
	var st controllerRebootState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatalf("unmarshal state file: %v", err)
	}
	return st
}
