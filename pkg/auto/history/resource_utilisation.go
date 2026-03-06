package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/resourceutilisationpb"
)

func (a *automation) collectResourceUtilisationChanges(ctx context.Context, source config.Source, payloads chan<- []byte) {
	client := gen.NewResourceUtilisationApiClient(a.clients.ClientConn())

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		stream, err := client.PullResourceUtilisation(ctx, &gen.PullResourceUtilisationRequest{
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
				payload, err := proto.Marshal(change.GetResourceUtilisation())
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
		resp, err := client.GetResourceUtilisation(ctx, &gen.GetResourceUtilisationRequest{
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
