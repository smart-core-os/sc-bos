package pgxalerts

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

func (s *Server) notifyAdd(name string, alert *alertpb.Alert) {
	// notify
	s.bus.Send(context.Background(), &alertpb.PullAlertsResponse_Change{
		Name:       name,
		Type:       typespb.ChangeType_ADD,
		ChangeTime: alert.CreateTime,
		NewValue:   alert,
	})
}

func (s *Server) notifyUpdate(name string, original *alertpb.Alert, updated *alertpb.Alert) int {
	return s.bus.Send(context.Background(), &alertpb.PullAlertsResponse_Change{
		Name:       name,
		Type:       typespb.ChangeType_UPDATE,
		ChangeTime: timestamppb.Now(),
		OldValue:   original,
		NewValue:   updated,
	})
}

func (s *Server) notifyRemove(name string, existing *alertpb.Alert) int {
	return s.bus.Send(context.Background(), &alertpb.PullAlertsResponse_Change{
		Name:       name,
		Type:       typespb.ChangeType_REMOVE,
		ChangeTime: timestamppb.Now(),
		OldValue:   existing,
	})
}
