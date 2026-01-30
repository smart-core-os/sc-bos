package enterleave

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/enterleave/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/enterleavesensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

type enterLeave struct {
	traits.UnimplementedEnterLeaveSensorApiServer
	client traits.EnterLeaveSensorApiClient
	names  []string

	model *occupancysensorpb.Model
}

func (f *feature) applyConfig(ctx context.Context, cfg config.Root) error {
	announce := f.announcer.Replace(ctx)
	logger := f.logger

	if len(cfg.EnterLeaveSensors) > 0 {
		group := &Group{logger: logger}

		if len(cfg.EnterLeaveSensors) > 0 {
			elServer := &enterLeave{
				model:  occupancysensorpb.NewModel(),
				names:  cfg.EnterLeaveSensors,
				client: traits.NewEnterLeaveSensorApiClient(f.clients.ClientConn()),
			}
			group.enterLeaveClients = append(group.enterLeaveClients, enterleavesensorpb.WrapApi(elServer))
		}
		announce.Announce(cfg.Name, node.HasTrait(trait.EnterLeaveSensor, node.WithClients(enterleavesensorpb.WrapApi(group))))
	}

	return nil
}

func (e *enterLeave) GetEnterLeaveEvent(ctx context.Context, _ *traits.GetEnterLeaveEventRequest) (*traits.EnterLeaveEvent, error) {

	enterCount := int32(0)
	leaveCount := int32(0)
	all := make([]*traits.EnterLeaveEvent, len(e.names))
	for i, name := range e.names {
		event, err := e.client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{
			Name: name,
		})
		if err != nil {
			return nil, err
		}
		all[i] = event

		enterCount += *event.EnterTotal
		leaveCount += *event.LeaveTotal
	}

	return &traits.EnterLeaveEvent{
		EnterTotal: &enterCount,
		LeaveTotal: &leaveCount,
	}, nil
}

func (e *enterLeave) PullEnterLeaveEvents(request *traits.PullEnterLeaveEventsRequest, server traits.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {

	ctx := server.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			enterCount := int32(0)
			leaveCount := int32(0)
			all := make([]*traits.EnterLeaveEvent, len(e.names))

			for i, name := range e.names {
				event, err := e.client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{
					Name: name,
				})
				if err != nil {
					return err
				}
				all[i] = event

				enterCount += *event.EnterTotal
				leaveCount += *event.LeaveTotal
			}

			var enterLeaveChanges []*traits.PullEnterLeaveEventsResponse_Change
			enterLeaveChanges = append(enterLeaveChanges, &traits.PullEnterLeaveEventsResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(time.Now()),
				EnterLeaveEvent: &traits.EnterLeaveEvent{
					EnterTotal: &enterCount,
					LeaveTotal: &leaveCount,
				},
			})

			err := server.Send(&traits.PullEnterLeaveEventsResponse{
				Changes: enterLeaveChanges,
			})

			if err != nil {
				return err
			}
		}
	}
}
