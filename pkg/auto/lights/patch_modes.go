package lights

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// ModePatches contributes patches for changing the state based on mode changes.
type ModePatches struct {
	name   deviceName
	client traits.ModeApiClient
	logger *zap.Logger
}

func (o *ModePatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	// remove our signal when we shouldn't be contributing anymore
	defer func() {
		changes <- clearModeValuesTransition(o.name)
	}()
	return pull.Changes[Patcher](ctx, o, changes, pull.WithLogger(o.logger.Named("mode")))
}

func (o *ModePatches) Pull(ctx context.Context, changes chan<- Patcher) error {
	stream, err := o.client.PullModeValues(ctx, &traits.PullModeValuesRequest{Name: o.name})
	if err != nil {
		return err
	}

	for {
		change, err := stream.Recv()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- (*pullModeValuesTransition)(change):
		}
	}
}

func (o *ModePatches) Poll(ctx context.Context, changes chan<- Patcher) error {
	res, err := o.client.GetModeValues(ctx, &traits.GetModeValuesRequest{Name: o.name})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- getModeValuesPatcher{o.name, res}:
		return nil
	}
}

type pullModeValuesTransition traits.PullModeValuesResponse

func (o *pullModeValuesTransition) Patch(s *ReadState) {
	r := (*traits.PullModeValuesResponse)(o)

	for _, change := range r.Changes {
		s.Modes = change.ModeValues
	}
}

type getModeValuesPatcher struct {
	name deviceName
	res  *traits.ModeValues
}

func (g getModeValuesPatcher) Patch(s *ReadState) {
	s.Modes = g.res
}

type clearModeValuesTransition string

func (c clearModeValuesTransition) Patch(s *ReadState) {
	s.Modes = nil
}
