package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/smart-core-os/sc-bos/supervisor/internal/config"
	"github.com/smart-core-os/sc-bos/supervisor/internal/install"
	"github.com/smart-core-os/sc-bos/supervisor/internal/server"
)

// installReleaseCmd is the name of the subcommand that installs a release from a local tarball.
const installReleaseCmd = "install-release"

func runInstallRelease(args []string) error {
	logCfg := zap.NewProductionConfig()
	logCfg.Encoding = "console"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := logCfg.Build()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()
	return installRelease(args, os.Stdout, os.Stderr, logger)
}

// installRelease runs the install-release subcommand with args, the arguments after its name. It installs
// the release tarball named in args into the releases root from the config, writes what changed to stdout,
// and writes warnings to stderr. It fails if the config doesn't select the freebsdrc installer, or the
// Supervisor has an unfinished update. Invalid flags exit the process, as they do for the server.
func installRelease(args []string, stdout, stderr io.Writer, logger *zap.Logger) error {
	flags := flag.NewFlagSet(installReleaseCmd, flag.ExitOnError)
	configPath := flags.String("config", config.DefaultPath, "path to the JSON config file; if absent, built-in defaults are used")
	restart := flags.Bool("restart", false, "restart the BOS service after installing")
	lockTimeout := flags.Duration("lock-timeout", 30*time.Second, "how long to wait for another update to release the releases root")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "usage: sc-bos-supervisor %s [flags] TARBALL\n", installReleaseCmd)
		flags.PrintDefaults()
	}
	_ = flags.Parse(args) // with ExitOnError, Parse exits rather than returning an error
	if flags.NArg() != 1 {
		flags.Usage()
		return errors.New("want exactly one release tarball")
	}
	tarball := flags.Arg(0)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if cfg.Installer != config.InstallerFreeBSDRC {
		return fmt.Errorf("%s needs the %q installer, but the config selects %q", installReleaseCmd, config.InstallerFreeBSDRC, cfg.Installer)
	}
	version, unfinished, err := server.UnfinishedUpdate(cfg.StateDir)
	switch {
	case errors.Is(err, server.ErrMalformedState):
		// The Supervisor treats a malformed record as empty, so there's no update to wait for.
		logger.Warn("ignoring the Supervisor's state file", zap.Error(err))
	case err != nil:
		return err
	}
	if unfinished {
		return fmt.Errorf("the Supervisor has an unfinished update to %s; try again once it finishes", version)
	}

	installer := install.NewFreeBSDRCInstaller(cfg.FreeBSDRC.Root, cfg.FreeBSDRC.Service, logger)
	ctx, cancel := context.WithTimeout(context.Background(), *lockTimeout)
	defer cancel()
	res, err := installer.ApplyLocal(ctx, tarball, *restart)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("an update is in progress: the releases root is still locked after %s", *lockTimeout)
	}
	if res.Version != "" {
		if res.OldCurrent == res.Version {
			_, _ = fmt.Fprintf(stdout, "%s is already current\n", res.Version)
		} else {
			_, _ = fmt.Fprintf(stdout, "installed %s; current -> %s (was: %s)\n", res.Version, res.Version, cmp.Or(res.OldCurrent, "none"))
		}
	}
	if err != nil {
		return err
	}
	if *restart {
		_, _ = fmt.Fprintf(stdout, "restarted %s\n", cfg.FreeBSDRC.Service)
	}
	// The Supervisor may have accepted an update while this command waited for the lock.
	if version, unfinished, err := server.UnfinishedUpdate(cfg.StateDir); err == nil && unfinished {
		_, _ = fmt.Fprintf(stderr, "warning: the Supervisor started an update to %s; current will change once it finishes\n", version)
	}
	// After a finished update, the Supervisor doesn't learn about a version installed by this command.
	if committed, settled, err := server.SettledCommit(cfg.StateDir); err == nil && settled && committed != "" && committed != res.Version {
		_, _ = fmt.Fprintf(stderr, "warning: the Supervisor still records %s as the committed version; "+
			"if its next update fails, the rollback waits for %s to commit and reports it as not confirmed\n", committed, committed)
	}
	return nil
}
