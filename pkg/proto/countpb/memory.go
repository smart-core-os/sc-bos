package countpb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type MemoryDevice struct {
	UnimplementedCountApiServer

	count *resource.Value
}

// compile time check that we implement the interface we need
var _ CountApiServer = (*MemoryDevice)(nil)

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		count: resource.NewValue(
			resource.WithInitialValue(InitialCount()),
			resource.WithWritablePaths(&Count{}, "added", "removed"),
		),
	}
}

func InitialCount() *Count {
	return &Count{
		ResetTime: timestamppb.Now(),
	}
}

func (t *MemoryDevice) GetCount(_ context.Context, req *GetCountRequest) (*Count, error) {
	return t.count.Get(resource.WithReadMask(req.ReadMask)).(*Count), nil
}

func (t *MemoryDevice) ResetCount(_ context.Context, request *ResetCountRequest) (*Count, error) {
	rt := request.ResetTime
	if rt == nil {
		rt = timestamppb.Now()
	}
	res, err := t.count.Set(&Count{Added: 0, Removed: 0, ResetTime: rt}, resource.WithAllFieldsWritable())
	return res.(*Count), err
}

func (t *MemoryDevice) UpdateCount(_ context.Context, request *UpdateCountRequest) (*Count, error) {
	res, err := t.count.Set(request.Count, resource.WithUpdateMask(request.UpdateMask), resource.InterceptBefore(func(old, value proto.Message) {
		if request.Delta {
			tOld := old.(*Count)
			tValue := value.(*Count)
			tValue.Added += tOld.Added
			tValue.Removed += tOld.Removed
		}
	}))
	return res.(*Count), err
}

func (t *MemoryDevice) PullCounts(request *PullCountsRequest, server CountApi_PullCountsServer) error {
	for event := range t.count.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullCountsResponse_Change{
			Name:  request.Name,
			Count: event.Value.(*Count),
		}
		err := server.Send(&PullCountsResponse{
			Changes: []*PullCountsResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
