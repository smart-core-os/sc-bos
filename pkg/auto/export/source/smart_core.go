package source

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/export/config"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func NewSmartCore(services Services) task.Starter {
	r := &smartCore{services: services}
	r.Lifecycle = task.NewLifecycle(r.applyConfig)
	r.Logger = services.Logger.Named("smart-core")
	return r
}

type smartCore struct {
	*task.Lifecycle[config.SmartCoreSource]
	services Services
}

func (s *smartCore) applyConfig(ctx context.Context, cfg config.SmartCoreSource) error {
	conn := s.services.Node.ClientConn()
	parentClient := traits.NewParentApiClient(conn)
	lightClient := traits.NewLightApiClient(conn)

	sent := allowDuplicates()
	if cfg.Duplicates.TrackDuplicates() {
		sent = trackDuplicates(cfg.Duplicates.Cmp())
	}

	children, err := parentClient.ListChildren(ctx, &traits.ListChildrenRequest{})
	if err != nil {
		return err
	}

	// todo: support better error handling for these subscriptions.
	// With this code any subscription or any publication that fails will fail the entire group
	tasks, ctx := errgroup.WithContext(ctx)
	for _, child := range children.Children {
		name := child.Name
		logger := s.Logger.With(zap.String("child", name))
		for _, traitProto := range child.Traits {
			traitName := trait.Name(traitProto.Name)
			logger := logger.With(zap.Stringer("trait", traitName))

			switch traitName {
			case trait.Light:
				tasks.Go(func() error {
					return publishLightBrightness(ctx, name, lightClient, s.services.Publisher, sent, logger)
				})
			}
		}
	}

	go func() {
		err := tasks.Wait()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			s.Logger.Warn("source shut down", zap.Error(err))
		} else {
			s.Logger.Debug("source shut down")
		}
	}()

	return nil
}

func publishLightBrightness(ctx context.Context, name string, lightClient traits.LightApiClient, publisher Publisher, sent *duplicates, logger *zap.Logger) error {
	puller := &lightBrightnessPuller{
		client: lightClient,
		name:   name,
	}
	changes := make(chan *traits.PullBrightnessResponse_Change)
	tasks, ctx := errgroup.WithContext(ctx)
	tasks.Go(func() error {
		defer close(changes)
		err := pull.Changes[*traits.PullBrightnessResponse_Change](ctx, puller, changes, pull.WithLogger(logger))
		if status.Code(err) == codes.Unimplemented {
			logger.Debug("read not supported")
			return nil
		}
		return err
	})
	tasks.Go(func() error {
		for change := range changes {
			if commit, publish := sent.Changed(name, change.Brightness); publish {
				data, err := protojson.MarshalOptions{
					EmitUnpopulated: true,
				}.Marshal(change.Brightness)
				if err != nil {
					return err
				}
				err = publisher.Publish(ctx, name, string(data))
				if err != nil {
					return err
				}
				commit()
			}
		}
		return nil
	})
	return tasks.Wait()
}

type lightBrightnessPuller struct {
	client traits.LightApiClient
	name   string
}

func (p *lightBrightnessPuller) Pull(ctx context.Context, changes chan<- *traits.PullBrightnessResponse_Change) error {
	stream, err := p.client.PullBrightness(ctx, &traits.PullBrightnessRequest{Name: p.name})
	if err != nil {
		return err
	}

	for {
		change, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, item := range change.Changes {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case changes <- item:
			}
		}

	}
}

func (p *lightBrightnessPuller) Poll(ctx context.Context, changes chan<- *traits.PullBrightnessResponse_Change) error {
	res, err := p.client.GetBrightness(ctx, &traits.GetBrightnessRequest{Name: p.name})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- &traits.PullBrightnessResponse_Change{Name: p.name, Brightness: res, ChangeTime: timestamppb.Now()}:
		return nil
	}
}
