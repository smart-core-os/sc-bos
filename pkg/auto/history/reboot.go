package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/rebootpb"
)

func (a *automation) collectRebootEventChanges(ctx context.Context, source config.Source, payloads chan<- []byte) {
	client := rebootpb.NewRebootApiClient(a.clients.ClientConn())

	// Deduplicate on boot_time: a new boot_time means a new boot event.
	// We use proto equality (cmp.Equal with no options) so that any change
	// to the RebootState triggers a record, but reconnects to the same boot
	// session are ignored.
	last := newDeduper[*rebootpb.RebootState](nil)

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		stream, err := client.PullRebootState(ctx, &rebootpb.PullRebootStateRequest{Name: source.Name, UpdatesOnly: false})
		if err != nil {
			return err
		}
		for {
			msg, err := stream.Recv()
			if err != nil {
				return err
			}
			for _, change := range msg.Changes {
				if !last.Changed(change.GetRebootState()) {
					continue
				}
				payload, err := proto.Marshal(change.GetRebootState())
				if err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case changes <- payload:
				}
			}
		}
	}

	pollFn := func(ctx context.Context, changes chan<- []byte) error {
		resp, err := client.GetRebootState(ctx, &rebootpb.GetRebootStateRequest{Name: source.Name})
		if err != nil {
			return err
		}
		if !last.Changed(resp) {
			return nil
		}
		payload, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- payload:
		}
		return nil
	}

	if err := collectChanges(ctx, source, pullFn, pollFn, payloads, a.logger); err != nil {
		a.logger.Warn("collection aborted", zap.Error(err))
	}
}
