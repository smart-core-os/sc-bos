package history

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
)

func (a *automation) collectLockerChanges(ctx context.Context, source config.Source, payloads chan<- []byte) {
	client := gen.NewUnlockableAPIClient(a.clients.ClientConn())

	var dedupes []*deduper[*gen.Unlockable]

	pullFn := func(ctx context.Context, changes chan<- []byte) error {
		return status.Error(codes.Unimplemented, "pull not implemented for unlockable changes")
	}

	pollFn := func(ctx context.Context, changes chan<- []byte) error {
		resp, err := client.ListUnlockables(ctx, &gen.ListUnlockablesRequest{Name: source.Name})

		if err != nil {
			return err
		}

		count := 0

		for _, bank := range resp.GetUnlockableBanks() {
			for _, unlockable := range bank.GetUnlockables() {
				if len(dedupes) <= count || dedupes[count] == nil {
					dedupes = append(dedupes, newDeduper[*gen.Unlockable](cmp.Equal()))
				}

				if !dedupes[count].Changed(unlockable) {
					count++
					continue
				}

				count++
				// there is a new record for each unlockable that has changed on this device name
				// (in this case the device is an unlockable bank service)
				payload, err := proto.Marshal(unlockable)

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
		return nil
	}

	if err := collectChanges(ctx, source, pullFn, pollFn, payloads, a.logger); err != nil {
		a.logger.Warn("collection aborted", zap.Error(err))
	}
}
