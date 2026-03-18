package lights

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// BrightnessSensorPatches contributes patches for changing the state based on brightness sensor readings.
type BrightnessSensorPatches struct {
	name   deviceName
	client brightnesssensorpb.BrightnessSensorApiClient
	logger *zap.Logger
}

func (o *BrightnessSensorPatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	// remove our signal when we shouldn't be contributing anymore
	defer func() {
		changes <- clearAmbientBrightnessTransition(o.name)
	}()
	return pull.Changes[Patcher](ctx, o, changes, pull.WithLogger(o.logger.Named("brightness")))
}

func (o *BrightnessSensorPatches) Pull(ctx context.Context, changes chan<- Patcher) error {
	stream, err := o.client.PullAmbientBrightness(ctx, &brightnesssensorpb.PullAmbientBrightnessRequest{Name: o.name})
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
		case changes <- (*pullAmbientBrightnessTransition)(change):
		}
	}
}

func (o *BrightnessSensorPatches) Poll(ctx context.Context, changes chan<- Patcher) error {
	res, err := o.client.GetAmbientBrightness(ctx, &brightnesssensorpb.GetAmbientBrightnessRequest{Name: o.name})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- getAmbientBrightnessPatcher{o.name, res}:
		return nil
	}
}

type pullAmbientBrightnessTransition brightnesssensorpb.PullAmbientBrightnessResponse

func (o *pullAmbientBrightnessTransition) Patch(s *ReadState) {
	r := (*brightnesssensorpb.PullAmbientBrightnessResponse)(o)

	for _, change := range r.Changes {
		s.AmbientBrightness[change.Name] = change.AmbientBrightness
	}
}

type getAmbientBrightnessPatcher struct {
	name deviceName
	res  *brightnesssensorpb.AmbientBrightness
}

func (g getAmbientBrightnessPatcher) Patch(s *ReadState) {
	s.AmbientBrightness[g.name] = g.res
}

type clearAmbientBrightnessTransition string

func (c clearAmbientBrightnessTransition) Patch(s *ReadState) {
	delete(s.AmbientBrightness, string(c))
}
