package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
)

func (a *automation) collectEnterLeaveEventChanges(ctx context.Context, source config.Source, payloads chan<- []byte) {
	client := traits.NewEnterLeaveSensorApiClient(a.clients.ClientConn())

	last := newDeduper[*traits.EnterLeaveEvent](cmp.Equal(cmp.FloatValueApprox(0, 0.0001)))

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		stream, err := client.PullEnterLeaveEvents(ctx, &traits.PullEnterLeaveEventsRequest{Name: source.Name, UpdatesOnly: true, ReadMask: source.ReadMask.PB()})
		if err != nil {
			return err
		}
		for {
			msg, err := stream.Recv()
			if err != nil {
				return err
			}
			for _, change := range msg.Changes {
				if !last.Changed(change.GetEnterLeaveEvent()) {
					continue
				}

				payload, err := proto.Marshal(change.GetEnterLeaveEvent())
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
		resp, err := client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: source.Name, ReadMask: source.ReadMask.PB()})
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
