package emergencypb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type MemoryDevice struct {
	UnimplementedEmergencyApiServer
	state *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		state: resource.NewValue(resource.WithInitialValue(InitialEmergency())),
	}
}

func InitialEmergency() *Emergency {
	return &Emergency{
		Level:           Emergency_OK,
		LevelChangeTime: serverTimestamp(),
	}
}

func (t *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	RegisterEmergencyApiServer(server, t)
}

func (t *MemoryDevice) GetEmergency(_ context.Context, req *GetEmergencyRequest) (*Emergency, error) {
	return t.state.Get(resource.WithReadMask(req.ReadMask)).(*Emergency), nil
}

func (t *MemoryDevice) UpdateEmergency(_ context.Context, request *UpdateEmergencyRequest) (*Emergency, error) {
	update, err := t.state.Set(request.Emergency, resource.WithUpdateMask(request.UpdateMask), resource.InterceptAfter(func(old, new proto.Message) {
		// user server time if the level changed but the change time didn't
		oldt, newt := old.(*Emergency), new.(*Emergency)
		if newt.Level != oldt.Level && oldt.LevelChangeTime == newt.LevelChangeTime {
			newt.LevelChangeTime = serverTimestamp()
		}
	}))
	return update.(*Emergency), err
}

func (t *MemoryDevice) PullEmergency(request *PullEmergencyRequest, server EmergencyApi_PullEmergencyServer) error {
	for event := range t.state.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullEmergencyResponse_Change{
			Name:       request.Name,
			Emergency:  event.Value.(*Emergency),
			ChangeTime: timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&PullEmergencyResponse{
			Changes: []*PullEmergencyResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
