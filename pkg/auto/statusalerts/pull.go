package statusalerts

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

func pullFrom(ctx context.Context, name string, client statuspb.StatusApiClient, c chan<- *statuspb.StatusLog) error {
	puller := statusLogPuller{client: client, name: name}
	return pull.Changes[*statuspb.StatusLog](ctx, puller, c)
}

type statusLogPuller struct {
	client statuspb.StatusApiClient
	name   string
}

func (s statusLogPuller) Pull(ctx context.Context, changes chan<- *statuspb.StatusLog) error {
	stream, err := s.client.PullCurrentStatus(ctx, &statuspb.PullCurrentStatusRequest{Name: s.name})
	if err != nil {
		return err
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range res.Changes {
			if err := chans.SendContext(ctx, changes, change.CurrentStatus); err != nil {
				return err
			}
		}
	}
}

func (s statusLogPuller) Poll(ctx context.Context, changes chan<- *statuspb.StatusLog) error {
	status, err := s.client.GetCurrentStatus(ctx, &statuspb.GetCurrentStatusRequest{Name: s.name})
	if err != nil {
		return err
	}
	return chans.SendContext(ctx, changes, status)
}
