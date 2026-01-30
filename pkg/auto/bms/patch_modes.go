package bms

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// ModePatches contributes patches for changing the state based on mode changes.
// Note that read times are recorded against _unique mode value updates_, not against sends from the remote device.
// This helps with tracking as we can see when a mode key last changed its value, this is more important than when any mode key changed its value.
type ModePatches struct {
	name   string
	client traits.ModeApiClient
	logger *zap.Logger
}

func (o *ModePatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	// remove our signal when we shouldn't be contributing anymore
	defer func() {
		changes <- clearModeValuesTransition(o.name)
	}()
	return pull.Changes[Patcher](ctx, o, changes, pull.WithLogger(o.logger))
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
		saveModeChange(s, change.Name, change.ModeValues.Values)
	}
}

type getModeValuesPatcher struct {
	name string
	res  *traits.ModeValues
}

func (g getModeValuesPatcher) Patch(s *ReadState) {
	saveModeChange(s, g.name, g.res.Values)
}

func saveModeChange(s *ReadState, name string, values map[string]string) {
	save, ok := s.Modes[name]
	if !ok {
		save = make(map[string]Value[string])
		s.Modes[name] = save
	}
	for k, v := range values {
		cv := save[k]
		if cv.V == v {
			continue
		}
		cv.set(s.Now(), v, nil)
		save[k] = cv
	}
}

type clearModeValuesTransition string

func (c clearModeValuesTransition) Patch(s *ReadState) {
	delete(s.Modes, string(c))
}
