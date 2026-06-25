// Package supervisor provides a client for the Supervisor's gRPC API, which BOS calls to manage
// software updates. The Supervisor is a separate host process that installs updates out-of-process
// and rolls them back automatically if the new version is unhealthy.
//
// The client connects over a Unix socket; the decision to skip calls when the feature is disabled
// belongs to the caller (Phase 1 wiring), not this package.
package supervisor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// Client is a thin wrapper over supervisorpb.SupervisorApiClient that dials the Supervisor's
// Unix-socket gRPC API and applies a per-call timeout to each RPC.
type Client struct {
	api     supervisorpb.SupervisorApiClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

// Dial creates a Client connected to the Supervisor's Unix socket at socketPath.
// grpc.NewClient is lazy, so the socket need not exist at the time of this call.
// timeout is applied to each RPC; pass 0 to use no per-call deadline.
func Dial(socketPath string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(
		"unix:"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		api:     supervisorpb.NewSupervisorApiClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

// Commit asserts to the Supervisor that version is the currently running, healthy BOS version.
// It must be called on every healthy startup; the Supervisor rolls back if no matching Commit
// arrives within its deadline. With no update in flight it acts as a routine heartbeat that
// records version as the rollback baseline for the next update.
func (c *Client) Commit(ctx context.Context, version string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.Commit(ctx, &supervisorpb.CommitRequest{Version: version})
	return err
}

// InstallUpdate asks the Supervisor to download and install the given update.
// The call returns as soon as the request is accepted; the install runs asynchronously because
// applying it recreates the BOS container and severs this connection.
// It returns a codes.FailedPrecondition status error if an update is already in progress.
func (c *Client) InstallUpdate(ctx context.Context, version, downloadURL, sha256 string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.InstallUpdate(ctx, &supervisorpb.InstallUpdateRequest{
		Version:     version,
		DownloadUrl: downloadURL,
		Sha256:      sha256,
	})
	return err
}

// GetUpdateStatus returns the state of the most recent or in-progress update.
func (c *Client) GetUpdateStatus(ctx context.Context) (*supervisorpb.UpdateStatus, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.GetUpdateStatus(ctx, &supervisorpb.GetUpdateStatusRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetStatus(), nil
}

// InstallSupervisorUpdate asks the Supervisor to update ITSELF to version, packaged as an RPM at
// downloadURL. Like InstallUpdate it returns once accepted; the install runs out-of-process and the
// Supervisor restarts onto the new binary (or rolls back). It returns a codes.FailedPrecondition status
// error if a self-update is already in progress.
func (c *Client) InstallSupervisorUpdate(ctx context.Context, version, downloadURL, sha256 string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.InstallSupervisorUpdate(ctx, &supervisorpb.InstallSupervisorUpdateRequest{
		Version:     version,
		DownloadUrl: downloadURL,
		Sha256:      sha256,
	})
	return err
}

// SupervisorInfo returns the running Supervisor's own version and the state of its most recent
// self-update (used to relay the self-update outcome to Smart Core Connect).
func (c *Client) SupervisorInfo(ctx context.Context) (version string, selfUpdate *supervisorpb.UpdateStatus, err error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.GetSupervisorInfo(ctx, &supervisorpb.GetSupervisorInfoRequest{})
	if err != nil {
		return "", nil, err
	}
	return resp.GetVersion(), resp.GetSelfUpdate(), nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// withTimeout wraps ctx with the client's per-call timeout, if one is configured.
// The returned cancel must always be called.
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}
