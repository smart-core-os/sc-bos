package boot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bootpb"
)

// newTestSystem creates a minimal System with a fresh temp data dir and a no-op requestReboot.
// Returns both the System and its backing node so callers can create clients via node.ClientConn().
func newTestSystem(t *testing.T) (*System, *node.Node) {
	t.Helper()
	n := node.New("test-node")
	s := &System{
		nodeName:      "test-node",
		announcer:     node.NewReplaceAnnouncer(n),
		logger:        zaptest.NewLogger(t),
		dataDir:       t.TempDir(),
		requestReboot: func() {},
	}
	return s, n
}

// startApplyConfig runs s.applyConfig in a background goroutine and registers a cleanup that
// cancels the context and waits for the goroutine to exit.
func startApplyConfig(t *testing.T, s *System) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.applyConfig(ctx, config{})
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// waitBootState polls GetBootState (via node loopback) until the route is registered and returns
// the current BootState. It signals that applyConfig has fully announced.
func waitBootState(t *testing.T, n *node.Node, name string) *bootpb.BootState {
	t.Helper()
	cli := bootpb.NewBootApiClient(n.ClientConn())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		bs, err := cli.GetBootState(t.Context(), &bootpb.GetBootStateRequest{Name: name})
		if err == nil {
			return bs
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for boot state to become available")
	return nil
}

// --- State file helpers ---

func TestSystem_stateFile_roundtrip(t *testing.T) {
	s, _ := newTestSystem(t)
	want := RebootState{Reason: "scheduled-maintenance", CleanExit: true}
	if err := WriteStateFile(s.dataDir, want); err != nil {
		t.Fatalf("WriteStateFile: %v", err)
	}
	got, err := ReadStateFile(s.dataDir)
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if got.Reason != want.Reason || got.CleanExit != want.CleanExit {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSystem_stateFile_withActor(t *testing.T) {
	s, _ := newTestSystem(t)
	actor := &actorpb.Actor{DisplayName: "alice"}
	actorJSON, err := protojson.Marshal(actor)
	if err != nil {
		t.Fatalf("marshal actor: %v", err)
	}
	want := RebootState{Reason: "actor-test", CleanExit: true, Actor: actorJSON}
	if err := WriteStateFile(s.dataDir, want); err != nil {
		t.Fatalf("WriteStateFile: %v", err)
	}
	got, err := ReadStateFile(s.dataDir)
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if got.Reason != want.Reason || got.CleanExit != want.CleanExit {
		t.Errorf("got %+v, want %+v", got, want)
	}
	var gotActor actorpb.Actor
	if err := protojson.Unmarshal(got.Actor, &gotActor); err != nil {
		t.Fatalf("unmarshal actor: %v", err)
	}
	if diff := cmp.Diff(actor, &gotActor, protocmp.Transform()); diff != "" {
		t.Errorf("actor mismatch (-want +got):\n%s", diff)
	}
}

func TestSystem_stateFile_missing(t *testing.T) {
	s, _ := newTestSystem(t)
	_, err := ReadStateFile(s.dataDir)
	if err == nil {
		t.Error("ReadStateFile: expected error for missing file, got nil")
	}
}

func TestSystem_stateFile_malformed(t *testing.T) {
	s, _ := newTestSystem(t)
	if err := os.WriteFile(filepath.Join(s.dataDir, "reboot-state.json"), []byte("not-json{"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := ReadStateFile(s.dataDir)
	if err == nil {
		t.Error("ReadStateFile: expected error for malformed JSON, got nil")
	}
}

func TestSystem_writeStateFile_emptyDataDir(t *testing.T) {
	if err := WriteStateFile("", RebootState{CleanExit: true}); err != nil {
		t.Errorf("WriteStateFile with empty dataDir: %v", err)
	}
}

// --- onReboot ---

func TestSystem_onReboot_graceful(t *testing.T) {
	called := make(chan struct{}, 1)
	s, _ := newTestSystem(t)
	s.requestReboot = func() { called <- struct{}{} }

	req := &bootpb.RebootRequest{Reason: "graceful-test", Force: false}
	if err := s.onReboot(t.Context(), req); err != nil {
		t.Fatalf("onReboot: %v", err)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("requestReboot was not called")
	}

	st, err := ReadStateFile(s.dataDir)
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if st.Reason != "graceful-test" || !st.CleanExit {
		t.Errorf("state = %+v, want Reason=%q CleanExit=true", st, "graceful-test")
	}
}

func TestSystem_onReboot_withActor(t *testing.T) {
	s, _ := newTestSystem(t)
	actor := &actorpb.Actor{DisplayName: "alice"}
	req := &bootpb.RebootRequest{Reason: "actor-test", Actor: actor, Force: false}
	if err := s.onReboot(t.Context(), req); err != nil {
		t.Fatalf("onReboot: %v", err)
	}

	st, err := ReadStateFile(s.dataDir)
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if len(st.Actor) == 0 {
		t.Fatal("actor not written to state file")
	}
	var got actorpb.Actor
	if err := protojson.Unmarshal(st.Actor, &got); err != nil {
		t.Fatalf("unmarshal actor: %v", err)
	}
	if diff := cmp.Diff(actor, &got, protocmp.Transform()); diff != "" {
		t.Errorf("actor mismatch (-want +got):\n%s", diff)
	}
}

func TestSystem_onReboot_stateWrittenBeforeReboot(t *testing.T) {
	s, _ := newTestSystem(t)
	var stateOK bool
	s.requestReboot = func() {
		st, err := ReadStateFile(s.dataDir)
		stateOK = err == nil && st.Reason == "order-test" && st.CleanExit
	}

	req := &bootpb.RebootRequest{Reason: "order-test", Force: false}
	if err := s.onReboot(t.Context(), req); err != nil {
		t.Fatalf("onReboot: %v", err)
	}
	if !stateOK {
		t.Error("state file not written (or incorrect) before requestReboot was called")
	}
}

// --- applyConfig ---

func TestSystem_applyConfig_firstBoot(t *testing.T) {
	// No prior state file → BootState has empty reason and no actor.
	s, n := newTestSystem(t)
	startApplyConfig(t, s)
	bs := waitBootState(t, n, "test-node")
	if bs.LastRebootReason != "" {
		t.Errorf("first boot: LastRebootReason = %q, want empty", bs.LastRebootReason)
	}
	if bs.LastRebootActor != nil {
		t.Errorf("first boot: LastRebootActor = %v, want nil", bs.LastRebootActor)
	}
}

func TestSystem_applyConfig_crashDetection(t *testing.T) {
	// State file exists but CleanExit is false (zero value) → detected as crash.
	s, n := newTestSystem(t)
	if err := WriteStateFile(s.dataDir, RebootState{}); err != nil { // CleanExit=false
		t.Fatalf("setup: %v", err)
	}
	startApplyConfig(t, s)
	bs := waitBootState(t, n, "test-node")
	const wantReason = "unexpected process exit"
	if bs.LastRebootReason != wantReason {
		t.Errorf("crash detection: LastRebootReason = %q, want %q", bs.LastRebootReason, wantReason)
	}
}

func TestSystem_applyConfig_cleanReboot(t *testing.T) {
	// State file has CleanExit=true and a reason → reason propagated to BootState.
	s, n := newTestSystem(t)
	if err := WriteStateFile(s.dataDir, RebootState{Reason: "scheduled maintenance", CleanExit: true}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	startApplyConfig(t, s)
	bs := waitBootState(t, n, "test-node")
	if bs.LastRebootReason != "scheduled maintenance" {
		t.Errorf("clean reboot: LastRebootReason = %q, want %q", bs.LastRebootReason, "scheduled maintenance")
	}
}

func TestSystem_applyConfig_actorRoundTrip(t *testing.T) {
	// Actor written by onReboot survives restart via state file.
	s, n := newTestSystem(t)
	actor := &actorpb.Actor{DisplayName: "operator"}
	actorJSON, err := protojson.Marshal(actor)
	if err != nil {
		t.Fatalf("marshal actor: %v", err)
	}
	if err := WriteStateFile(s.dataDir, RebootState{Reason: "operator-restart", CleanExit: true, Actor: actorJSON}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	startApplyConfig(t, s)
	bs := waitBootState(t, n, "test-node")
	if bs.LastRebootReason != "operator-restart" {
		t.Errorf("actor round-trip: LastRebootReason = %q, want %q", bs.LastRebootReason, "operator-restart")
	}
	if diff := cmp.Diff(actor, bs.LastRebootActor, protocmp.Transform()); diff != "" {
		t.Errorf("actor round-trip: LastRebootActor mismatch (-want +got):\n%s", diff)
	}
}

func TestSystem_applyConfig_writesInProgressMarker(t *testing.T) {
	// applyConfig overwrites any prior state with the in-progress marker {} on startup.
	s, n := newTestSystem(t)
	if err := WriteStateFile(s.dataDir, RebootState{Reason: "old-reason", CleanExit: true}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	startApplyConfig(t, s)
	// waitBootState ensures applyConfig has announced (i.e., has already written the marker).
	waitBootState(t, n, "test-node")

	data, err := os.ReadFile(filepath.Join(s.dataDir, "reboot-state.json"))
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	var st RebootState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if st.Reason != "" || st.CleanExit {
		t.Errorf("in-progress marker: got %+v, want empty rebootState{}", st)
	}
}
