package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

// TestController_announceSoftwareVersion verifies the root node's metadata reports the
// running build version (via EffectiveVersion, controlled here by BOS_VERSION_OVERRIDE)
// as Product.SoftwareVersion, unless an operator pinned one in config.
func TestController_announceSoftwareVersion(t *testing.T) {
	const buildVersion = "v0.0.0-test-build"

	tests := []struct {
		name      string
		appConfig string
		want      string
	}{
		{
			name:      "fills build version when config omits it",
			appConfig: `{"name": "test-node"}`,
			want:      buildVersion,
		},
		{
			name:      "preserves software version pinned in config",
			appConfig: `{"name": "test-node", "metadata": {"product": {"software_version": "configured-1.0"}}}`,
			want:      "configured-1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BOS_VERSION_OVERRIDE", buildVersion)

			got := runAndReadSoftwareVersion(t, tt.appConfig)
			if got != tt.want {
				t.Errorf("Product.SoftwareVersion = %q, want %q", got, tt.want)
			}
		})
	}
}

// runAndReadSoftwareVersion bootstraps a controller from the given app config JSON, runs it
// long enough to announce the node's metadata, then reads back Product.SoftwareVersion via
// the in-process MetadataApi.
func runAndReadSoftwareVersion(t *testing.T, appConfig string) string {
	t.Helper()

	dir := t.TempDir()
	confPath := filepath.Join(dir, "app.conf.json")
	if err := os.WriteFile(confPath, []byte(appConfig), 0o644); err != nil {
		t.Fatalf("write app config: %v", err)
	}

	config := sysconf.Default()
	config.PolicyMode = sysconf.PolicyOff
	config.DataDir = dir
	config.AppConfig = []string{confPath}
	// Don't bind real ports while testing.
	config.ListenGRPC = ""
	config.ListenHTTPS = ""

	c, err := Bootstrap(t.Context(), config)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Run announces the node metadata near startup and then blocks; cancel once we've read it.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	runErr := make(chan error, 1)
	go func() { runErr <- c.Run(ctx) }()

	// PullMetadata streams the current value then updates, so it delivers the software version
	// once Run announces it without us having to poll.
	client := metadatapb.NewMetadataApiClient(c.Node.ClientConn())
	stream, err := client.PullMetadata(ctx, &metadatapb.PullMetadataRequest{Name: c.Node.Name()})
	if err != nil {
		t.Fatalf("PullMetadata() error = %v", err)
	}
	for {
		res, err := stream.Recv()
		if err != nil {
			t.Fatalf("software version not announced before stream closed: %v", err)
		}
		for _, change := range res.GetChanges() {
			if v := change.GetMetadata().GetProduct().GetSoftwareVersion(); v != "" {
				cancel()
				<-runErr
				return v
			}
		}
	}
}
