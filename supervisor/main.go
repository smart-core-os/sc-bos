// Command supervisor is the Smart Core BOS Supervisor: a privileged system service that installs BOS
// software updates out-of-process and rolls them back locally if the new version is unhealthy.
//
// It exposes the SupervisorApi gRPC service over a Unix socket for BOS to drive updates. See
// supervisor/docs/state-model.md for the design.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/supervisor/internal/config"
	"github.com/smart-core-os/sc-bos/supervisor/internal/install"
	"github.com/smart-core-os/sc-bos/supervisor/internal/server"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "/etc/sc-bos-supervisor/config.json", "path to the JSON config file; if absent, built-in defaults are used")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(filepath.Dir(cfg.Socket), 0o755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	if err := os.Remove(cfg.Socket); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	lis, err := net.Listen("unix", cfg.Socket)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Socket, err)
	}
	if err := os.Chmod(cfg.Socket, 0o660); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	installer := install.NewPodmanInstaller(cfg.ImageRepo, cfg.Unit, logger.Named("installer"))
	svc := server.New(installer, cfg.StateDir, http.DefaultClient, cfg.CommitDeadline.Duration, cfg.AllowInsecureDownloads, logger.Named("server"))
	// Drive the persisted goal to completion before accepting new requests: an install interrupted by a
	// previous crash is resumed (or rolled back) here, so the local auto-recovery fires even across a
	// Supervisor crash. A fresh InstallUpdate runs the same reconcile.
	svc.Reconcile()

	grpcServer := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(grpcServer, svc)

	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcServer.Serve(lis) }()
	logger.Info("supervisor listening", zap.String("socket", cfg.Socket))

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(15 * time.Second):
		logger.Warn("graceful stop timed out, forcing")
		grpcServer.Stop()
	}

	// Trigger a graceful stop of any background tasks the service is executing. Downloads are cancellable so should
	// stop pretty much immediately. But Stop will wait for BOS service restarts to complete before returning, so
	// as not to leave it in an undefined state.
	// Set an absolute maximum limit to wait based on the commit deadline - we should never need to wait this long
	// legitimately, so if we hit that second timeout that's probably a bug.
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*cfg.CommitDeadline.Duration)
	defer cancel()
	if err := svc.Stop(stopCtx); err != nil {
		logger.Warn("in-flight install did not finish before shutdown", zap.Error(err))
	}
	return nil
}
