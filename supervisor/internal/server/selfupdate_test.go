package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/supervisor/internal/selfupdate"
)

// fakeSelfUpdater is a server.SelfUpdater for exercising the self-update RPC handlers in isolation.
type fakeSelfUpdater struct {
	version    string
	installErr error
	installed  []string // versions passed to Install
	last       *selfupdate.State
}

func (f *fakeSelfUpdater) Install(_ context.Context, version, _, _ string) (selfupdate.State, error) {
	if f.installErr != nil {
		return selfupdate.State{}, f.installErr
	}
	f.installed = append(f.installed, version)
	return selfupdate.State{Target: version, Phase: selfupdate.PhaseInstalling}, nil
}

func (f *fakeSelfUpdater) Version() string { return f.version }

func (f *fakeSelfUpdater) LastUpdate() (selfupdate.State, bool) {
	if f.last == nil {
		return selfupdate.State{}, false
	}
	return *f.last, true
}

func TestInstallSupervisorUpdate_Unconfigured(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil) // no SetSelfUpdater
	c := testClient(t, svc)

	_, err := c.InstallSupervisorUpdate(context.Background(), &supervisorpb.InstallSupervisorUpdateRequest{
		Version: "v2", DownloadUrl: "https://x/v2.rpm", Sha256: "ab",
	})
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestInstallSupervisorUpdate_Accepts(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil)
	fake := &fakeSelfUpdater{version: "v1"}
	svc.SetSelfUpdater(fake)
	c := testClient(t, svc)

	resp, err := c.InstallSupervisorUpdate(context.Background(), &supervisorpb.InstallSupervisorUpdateRequest{
		Version: "v2", DownloadUrl: "https://x/v2.rpm", Sha256: "ab",
	})
	require.NoError(t, err)
	assert.Equal(t, supervisorpb.UpdateStatus_INSTALLING, resp.GetStatus().GetState())
	assert.Equal(t, "v2", resp.GetStatus().GetVersion())
	assert.Equal(t, []string{"v2"}, fake.installed)
}

func TestInstallSupervisorUpdate_Validation(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil)
	svc.SetSelfUpdater(&fakeSelfUpdater{version: "v1"})
	c := testClient(t, svc)

	_, err := c.InstallSupervisorUpdate(context.Background(), &supervisorpb.InstallSupervisorUpdateRequest{
		Version: "", DownloadUrl: "https://x", Sha256: "ab",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInstallSupervisorUpdate_InProgressIsFailedPrecondition(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil)
	svc.SetSelfUpdater(&fakeSelfUpdater{version: "v1", installErr: selfupdate.ErrInProgress})
	c := testClient(t, svc)

	_, err := c.InstallSupervisorUpdate(context.Background(), &supervisorpb.InstallSupervisorUpdateRequest{
		Version: "v2", DownloadUrl: "https://x/v2.rpm", Sha256: "ab",
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGetSupervisorInfo_ReportsVersionAndStatus(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil)
	finish := time.Now()
	svc.SetSelfUpdater(&fakeSelfUpdater{
		version: "v2",
		last:    &selfupdate.State{Target: "v2", Phase: selfupdate.PhaseCompleted, FinishTime: &finish},
	})
	c := testClient(t, svc)

	resp, err := c.GetSupervisorInfo(context.Background(), &supervisorpb.GetSupervisorInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t, "v2", resp.GetVersion())
	assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, resp.GetSelfUpdate().GetState())
}

func TestGetSupervisorInfo_IdleWhenNoUpdate(t *testing.T) {
	svc := newService(t, &fakeInstaller{}, t.TempDir(), nil)
	svc.SetSelfUpdater(&fakeSelfUpdater{version: "v1"})
	c := testClient(t, svc)

	resp, err := c.GetSupervisorInfo(context.Background(), &supervisorpb.GetSupervisorInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t, "v1", resp.GetVersion())
	assert.Equal(t, supervisorpb.UpdateStatus_IDLE, resp.GetSelfUpdate().GetState())
}
