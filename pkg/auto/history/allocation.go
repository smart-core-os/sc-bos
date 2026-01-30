package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
)

func (a *automation) collectAllocationChanges(ctx context.Context, source config.Source, payloads chan<- []byte) {
	client := allocationpb.NewAllocationApiClient(a.clients.ClientConn())

	dedupe := newDeduper[*allocationpb.Allocation](cmp.Equal())

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		stream, err := client.PullAllocation(ctx, &allocationpb.PullAllocationRequest{Name: source.Name, UpdatesOnly: true, ReadMask: source.ReadMask.PB()})

		if err != nil {
			return err
		}

		for {
			msg, err := stream.Recv()
			if err != nil {
				return err
			}
			for _, change := range msg.Changes {
				if !dedupe.Changed(change.GetAllocation()) {
					continue
				}

				payload, err := proto.Marshal(change.GetAllocation())
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
		resp, err := client.GetAllocation(ctx, &allocationpb.GetAllocationRequest{Name: source.Name, ReadMask: source.ReadMask.PB()})
		if err != nil {
			return err
		}

		if !dedupe.Changed(resp) {
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
