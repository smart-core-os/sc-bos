package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/api"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
)

var (
	flagListen   string
	flagDataPath string
)

func init() {
	flag.StringVar(&flagListen, "listen", "127.0.0.1:8080", "interface:port to listen on")
	flag.StringVar(&flagDataPath, "data", "cloudsim.db", "path to SQLite database")
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err = run(ctx, logger)
	_ = logger.Sync()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *zap.Logger) (err error) {
	dataStore, err := store.OpenStore(ctx, flagDataPath, logger)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { err = errors.Join(err, dataStore.Close()) }()

	lis, err := net.Listen("tcp", flagListen)
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", flagListen, err)
	}
	logger.Info("server listening", zap.String("address", lis.Addr().String()))

	mux := http.NewServeMux()
	apiServer := api.NewServer(dataStore, logger)
	apiServer.RegisterRoutes(mux)

	return serveContext(ctx, lis, mux, logger)
}

func serveContext(ctx context.Context, lis net.Listener, handler http.Handler, logger *zap.Logger) error {
	server := http.Server{
		Handler: handler,
	}

	errCh := make(chan error)
	go func() {
		err := server.Serve(lis)
		if errors.Is(err, http.ErrServerClosed) {
			// not considered a reportable error
			err = nil
		}
		errCh <- err
	}()
	<-ctx.Done()
	// extend the context by up to 10 seconds for shutdown
	shutdownTimeout := 10 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	logger.Info("shutting down http server", zap.Duration("timeout", shutdownTimeout))
	err := server.Shutdown(shutdownCtx)
	return errors.Join(err, <-errCh)
}
