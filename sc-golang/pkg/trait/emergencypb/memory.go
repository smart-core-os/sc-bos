package emergencypb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
)

type MemoryDevice struct {
	traits.UnimplementedEmergencyApiServer
	state *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		state: resource.NewValue(resource.WithInitialValue(InitialEmergency())),
	}
}

func InitialEmergency() *traits.Emergency {
	return &traits.Emergency{
		Level:           traits.Emergency_OK,
		LevelChangeTime: serverTimestamp(),
	}
}

func (t *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	traits.RegisterEmergencyApiServer(server, t)
}

func (t *MemoryDevice) GetEmergency(_ context.Context, req *traits.GetEmergencyRequest) (*traits.Emergency, error) {
	return t.state.Get(resource.WithReadMask(req.ReadMask)).(*traits.Emergency), nil
}

func (t *MemoryDevice) UpdateEmergency(_ context.Context, request *traits.UpdateEmergencyRequest) (*traits.Emergency, error) {
	update, err := t.state.Set(request.Emergency, resource.WithUpdateMask(request.UpdateMask), resource.InterceptAfter(func(old, new proto.Message) {
		// user server time if the level changed but the change time didn't
		oldt, newt := old.(*traits.Emergency), new.(*traits.Emergency)
		if newt.Level != oldt.Level && oldt.LevelChangeTime == newt.LevelChangeTime {
			newt.LevelChangeTime = serverTimestamp()
		}
	}))
	return update.(*traits.Emergency), err
}

func (t *MemoryDevice) PullEmergency(request *traits.PullEmergencyRequest, server traits.EmergencyApi_PullEmergencyServer) error {
	for event := range t.state.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullEmergencyResponse_Change{
			Name:       request.Name,
			Emergency:  event.Value.(*traits.Emergency),
			ChangeTime: timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&traits.PullEmergencyResponse{
			Changes: []*traits.PullEmergencyResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
