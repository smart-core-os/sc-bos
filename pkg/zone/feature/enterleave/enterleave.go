package enterleave

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/enterleave/config"
)

type enterLeave struct {
	enterleavesensorpb.UnimplementedEnterLeaveSensorApiServer
	client enterleavesensorpb.EnterLeaveSensorApiClient
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
				client: enterleavesensorpb.NewEnterLeaveSensorApiClient(f.clients.ClientConn()),
			}
			group.enterLeaveClients = append(group.enterLeaveClients, enterleavesensorpb.NewEnterLeaveSensorApiClient(wrap.ServerToClient(enterleavesensorpb.EnterLeaveSensorApi_ServiceDesc, elServer)))
		}
		announce.Announce(cfg.Name,
			node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(group)),
			node.HasTrait(trait.EnterLeaveSensor),
		)
	}

	return nil
}

func (e *enterLeave) GetEnterLeaveEvent(ctx context.Context, _ *enterleavesensorpb.GetEnterLeaveEventRequest) (*enterleavesensorpb.EnterLeaveEvent, error) {

	enterCount := int32(0)
	leaveCount := int32(0)
	all := make([]*enterleavesensorpb.EnterLeaveEvent, len(e.names))
	for i, name := range e.names {
		event, err := e.client.GetEnterLeaveEvent(ctx, &enterleavesensorpb.GetEnterLeaveEventRequest{
			Name: name,
		})
		if err != nil {
			return nil, err
		}
		all[i] = event

		enterCount += *event.EnterTotal
		leaveCount += *event.LeaveTotal
	}

	return &enterleavesensorpb.EnterLeaveEvent{
		EnterTotal: &enterCount,
		LeaveTotal: &leaveCount,
	}, nil
}

func (e *enterLeave) PullEnterLeaveEvents(request *enterleavesensorpb.PullEnterLeaveEventsRequest, server enterleavesensorpb.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {

	ctx := server.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			enterCount := int32(0)
			leaveCount := int32(0)
			all := make([]*enterleavesensorpb.EnterLeaveEvent, len(e.names))

			for i, name := range e.names {
				event, err := e.client.GetEnterLeaveEvent(ctx, &enterleavesensorpb.GetEnterLeaveEventRequest{
					Name: name,
				})
				if err != nil {
					return err
				}
				all[i] = event

				enterCount += *event.EnterTotal
				leaveCount += *event.LeaveTotal
			}

			var enterLeaveChanges []*enterleavesensorpb.PullEnterLeaveEventsResponse_Change
			enterLeaveChanges = append(enterLeaveChanges, &enterleavesensorpb.PullEnterLeaveEventsResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(time.Now()),
				EnterLeaveEvent: &enterleavesensorpb.EnterLeaveEvent{
					EnterTotal: &enterCount,
					LeaveTotal: &leaveCount,
				},
			})

			err := server.Send(&enterleavesensorpb.PullEnterLeaveEventsResponse{
				Changes: enterLeaveChanges,
			})

			if err != nil {
				return err
			}
		}
	}
}
