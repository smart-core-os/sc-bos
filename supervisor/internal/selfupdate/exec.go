package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// DNF installs and downgrades the Supervisor RPM with dnf. dnf resolves dependencies and, via the
// package's %systemd_postun_with_restart scriptlet, restarts the Supervisor onto the new binary.
type DNF struct {
	logger *zap.Logger
}

// NewDNF returns a dnf-backed PackageManager.
func NewDNF(logger *zap.Logger) *DNF {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DNF{logger: logger}
}

func (d *DNF) Install(ctx context.Context, rpmPath string) error {
	return d.run(ctx, "install", rpmPath)
}

func (d *DNF) Downgrade(ctx context.Context, rpmPath string) error {
	return d.run(ctx, "downgrade", rpmPath)
}

func (d *DNF) run(ctx context.Context, verb, rpmPath string) error {
	args := []string{"-y", verb, rpmPath}
	d.logger.Debug("exec", zap.String("cmd", "dnf"), zap.Strings("args", args))
	out, err := exec.CommandContext(ctx, "dnf", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf %s %s: %w: %s", verb, rpmPath, err, bytes.TrimSpace(out))
	}
	return nil
}

// SystemdRunLauncher launches the applier as a transient systemd service. Running it under systemd
// (PID 1), not the Supervisor's own unit, is what lets it survive the restart the RPM install triggers.
type SystemdRunLauncher struct {
	exe        string // path to the Supervisor binary (the applier is a subcommand of it)
	configPath string // forwarded so the applier reads the same config (socket, stateDir, ...)
	logger     *zap.Logger
}

// NewSystemdRunLauncher returns a Launcher that runs "<exe> self-update-apply -config <configPath>" in a
// transient unit named sc-bos-supervisor-self-update.
func NewSystemdRunLauncher(exe, configPath string, logger *zap.Logger) *SystemdRunLauncher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SystemdRunLauncher{exe: exe, configPath: configPath, logger: logger}
}

func (l *SystemdRunLauncher) Launch(ctx context.Context) error {
	args := []string{
		"--unit=sc-bos-supervisor-self-update",
		"--collect", // discard the transient unit once it exits, even on failure
		l.exe, "self-update-apply", "-config", l.configPath,
	}
	l.logger.Debug("exec", zap.String("cmd", "systemd-run"), zap.Strings("args", args))
	out, err := exec.CommandContext(ctx, "systemd-run", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemd-run: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// SocketConfirmer confirms a Supervisor version by dialling the Supervisor socket and polling
// GetSupervisorInfo until it reports the wanted version. A reachable socket reporting that version means
// the new binary actually came up and serves requests.
type SocketConfirmer struct {
	socket string
	poll   time.Duration
	logger *zap.Logger
}

// NewSocketConfirmer returns a Confirmer that polls the Supervisor at socketPath.
func NewSocketConfirmer(socketPath string, logger *zap.Logger) *SocketConfirmer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SocketConfirmer{socket: socketPath, poll: time.Second, logger: logger}
}

func (c *SocketConfirmer) Confirm(ctx context.Context, want string) error {
	conn, err := grpc.NewClient("unix:"+c.socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial supervisor socket: %w", err)
	}
	defer func() { _ = conn.Close() }()
	client := supervisorpb.NewSupervisorApiClient(conn)

	ticker := time.NewTicker(c.poll)
	defer ticker.Stop()
	for {
		got := c.probe(ctx, client)
		if got == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("supervisor did not report version %q before deadline (last saw %q): %w", want, got, ctx.Err())
		case <-ticker.C:
		}
	}
}

// probe asks the running Supervisor for its version, returning "" if it is not reachable yet (e.g. mid
// restart). A short per-call timeout keeps a hung dial from eating the whole confirm window.
func (c *SocketConfirmer) probe(ctx context.Context, client supervisorpb.SupervisorApiClient) string {
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	resp, err := client.GetSupervisorInfo(callCtx, &supervisorpb.GetSupervisorInfoRequest{})
	if err != nil {
		c.logger.Debug("probe supervisor version", zap.Error(err))
		return ""
	}
	return strings.TrimSpace(resp.GetVersion())
}
