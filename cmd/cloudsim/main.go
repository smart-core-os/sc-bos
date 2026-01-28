package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"go.uber.org/zap"
)

var (
	flagListen string
)

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

func run(ctx context.Context, logger *zap.Logger) error {
	lis, err := net.Listen("tcp", flagListen)
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", flagListen, err)
	}
	logger.Info("server listening", zap.String("address", lis.Addr().String()))
}
