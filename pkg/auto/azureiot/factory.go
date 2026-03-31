package azureiot

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/iothub"
	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const FactoryName = "azureiot"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	services.Logger = services.Logger.Named(FactoryName)
	a := &Auto{services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(ParseConfig))
	return a
}

func (f factory) ConfigBlocks() []block.Block {
	return Blocks
}

type Auto struct {
	*service.Service[Config]
	services auto.Services

	connsMu sync.Mutex
	conns   map[string]*grpc.ClientConn
}

func (a *Auto) applyConfig(ctx context.Context, cfg Config) error {
	a.connsMu.Lock()
	for _, conn := range a.conns {
		conn.Close()
	}
	a.conns = make(map[string]*grpc.ClientConn)
	a.connsMu.Unlock()

	if len(cfg.Devices) == 0 {
		a.services.Logger.Warn("no devices configured; no polling will happen")
		return nil
	}

	// load the group key from string or disk, but only if a device will need it
	var needsGroupKey bool // group key is required if a device lacks a connection string
	for _, deviceCfg := range cfg.Devices {
		if !deviceCfg.UsesConnectionString() {
			needsGroupKey = true
		}
	}
	var groupKey auth.SASKey
	if needsGroupKey {
		var err error
		groupKey, err = loadGroupKey(cfg)
		if err != nil {
			return fmt.Errorf("failed to load group key: %w", err)
		}
	}

	// the group of tasks that pull from device traits and publish to IoT Hub
	grp, ctx := errgroup.WithContext(ctx)
	for _, device := range cfg.Devices {
		devDialler, err := diallerFromConfig(device, cfg.IDScope, groupKey)
		if err != nil {
			a.services.Logger.Error("device not initialised due to invalid configuration", zap.String("device", device.Name), zap.Error(err))
			continue
		}

		stream := make(chan proto.Message)
		grp.Go(func() error {
			defer close(stream)
			return a.pullDevice(ctx, stream, device)
		})
		grp.Go(func() error {
			a.sendOutputMessages(ctx, devDialler, stream)
			return nil
		})
	}

	go func() {
		err := grp.Wait()
		switch {
		case err == nil:
			// if everything completed without error, then nothing did anything anyway
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			// context errors are expected when the service is shutting down
		default:
			a.services.Logger.Error("one or more background tasks stopped", zap.Error(err))
		}
	}()

	return nil
}

// sendOutputMessages sends messages from stream to IoT Hub.
// It blocks until stream is closed.
func (a *Auto) sendOutputMessages(ctx context.Context, dialler dialler, stream <-chan proto.Message) {
	var conn iothub.Conn
	backlog := list.New()
	// notSent records that a message couldn't be sent, it will be sent later (if possible)
	notSent := func(m any) {
		for backlog.Len() > DefaultBacklogSize {
			drop := backlog.Front()
			backlog.Remove(drop)
			a.services.Logger.Warn("dropping message due to full buffer", zap.Any("message", drop.Value))
		}
		backlog.PushBack(m)
	}
	// sendBacklog sends all messages that couldn't be sent the first time
	sendBacklog := func() error {
		for m := backlog.Front(); m != nil; m = m.Next() {
			err := conn.SendOutputMessage(ctx, m.Value)
			if err != nil {
				return err
			}
			backlog.Remove(m)
		}
		return nil
	}
	for msg := range stream {
		if conn == nil {
			var err error
			conn, err = dialler.Dial(ctx)
			if err != nil {
				a.services.Logger.Error("failed to dial IoT Hub", zap.Error(err))
				notSent(msg)
				continue
			}
		}
		err := sendBacklog()
		if err != nil {
			a.services.Logger.Warn("failed to send backlog message to IoT Hub", zap.Error(err))
			notSent(msg)
			continue
		}
		err = conn.SendOutputMessage(ctx, msg)
		if err != nil {
			a.services.Logger.Warn("failed to send message to IoT Hub", zap.Error(err))
			notSent(msg)
			continue
		}
	}
}

func loadGroupKey(cfg Config) (auth.SASKey, error) {
	if cfg.GroupKey != "" {
		return auth.ParseSASKey(cfg.GroupKey)
	}

	raw, err := os.ReadFile(cfg.GroupKeyFile)
	if err != nil {
		return nil, err
	}
	return auth.ParseSASKey(string(raw))
}
