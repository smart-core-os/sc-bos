package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
)

func (a *automation) collectResourceUseChanges(ctx context.Context, source config.Source, payloads chan<- []byte, rec *recency, logger *zap.Logger) {
	client := gen.NewResourceUseApiClient(a.clients.ClientConn())

	// Every reading is recorded, so there is no deduper.
	// Each is still accepted on rec, so a heartbeat only comes once the source has gone quiet.

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		stream, err := client.PullResourceUse(ctx, &gen.PullResourceUseRequest{
			Name:        source.Name,
			UpdatesOnly: true,
			ReadMask:    source.ReadMask.PB(),
		})
		if err != nil {
			return err
		}
		for {
			msg, err := stream.Recv()
			if err != nil {
				return err
			}
			for _, change := range msg.Changes {
				payload, err := proto.Marshal(change.GetResourceUse())
				if err != nil {
					return err
				}
				if !rec.accept(ctx) {
					continue
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
		resp, err := client.GetResourceUse(ctx, &gen.GetResourceUseRequest{
			Name:     source.Name,
			ReadMask: source.ReadMask.PB(),
		})
		if err != nil {
			return err
		}
		payload, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		if !rec.accept(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- payload:
		}
		return nil
	}

	if err := collectChanges(ctx, source, pullFn, pollFn, payloads, rec, logger); err != nil {
		logger.Warn("collection aborted", zap.Error(err))
	}
}
