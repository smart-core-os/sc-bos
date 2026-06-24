package cloud

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

// stubConfigClient is a Client that serves a fixed payload and records every check-in, for driving
// installConfig in isolation from a real server.
type stubConfigClient struct {
	payload  []byte
	checkIns []CheckInRequest
}

func (c *stubConfigClient) CheckIn(_ context.Context, req CheckInRequest) (CheckInResponse, error) {
	c.checkIns = append(c.checkIns, req)
	return CheckInResponse{}, nil
}

func (c *stubConfigClient) DownloadPayload(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(c.payload)), nil
}

func (c *stubConfigClient) Renew(context.Context) (*Registration, error) { return nil, nil }

func (c *stubConfigClient) SetRegistration(*Registration) {}

// TestConfigFlow_MissingChecksumFailsTerminally locks: a config offer with no checksum is a
// non-retryable contract violation, so installConfig reports the deployment failed (not installing)
// and stages nothing - matching the binary channel's treatment of an invalid artefact.
func TestConfigFlow_MissingChecksumFailsTerminally(t *testing.T) {
	ctx := context.Background()
	storeDir, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = storeDir.Close() })
	store := NewDeploymentStore(storeDir)

	client := &stubConfigClient{payload: txtarToTarGZ(t, "single.txtar")}
	u := NewConfigUpdater(store, client)

	latest := &LatestStream{
		Deployment: StreamDeployment{ID: "c-7"},
		Version:    VersionProjection{ID: "7", PayloadURL: "https://example.test/payload"},
		// Checksum deliberately empty.
	}
	needReboot, err := u.installConfig(ctx, latest)
	if err == nil {
		t.Fatal("installConfig: want error for missing checksum, got nil")
	}
	if needReboot {
		t.Error("needReboot = true, want false when the checksum is missing")
	}

	// Nothing was staged.
	if id, err := store.InstallingID(); err != nil || id != "" {
		t.Errorf("InstallingID = %q, %v; want empty (nothing staged)", id, err)
	}

	// The last report drives the deployment terminally failed, not installing.
	if len(client.checkIns) == 0 {
		t.Fatal("expected at least one check-in")
	}
	last := client.checkIns[len(client.checkIns)-1]
	if len(last.Progress) != 1 {
		t.Fatalf("last check-in progress = %+v, want a single report", last.Progress)
	}
	p := last.Progress[0]
	if p.State != ProgressFailed {
		t.Errorf("progress state = %q, want %q", p.State, ProgressFailed)
	}
	if p.DeploymentID != "c-7" {
		t.Errorf("progress deployment id = %q, want %q", p.DeploymentID, "c-7")
	}
	if p.Reason == "" {
		t.Error("failed report has empty reason")
	}
}
