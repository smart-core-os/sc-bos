package countpb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type MemoryDevice struct {
	traits.UnimplementedCountApiServer

	count *resource.Value
}

// compile time check that we implement the interface we need
var _ traits.CountApiServer = (*MemoryDevice)(nil)

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		count: resource.NewValue(
			resource.WithInitialValue(InitialCount()),
			resource.WithWritablePaths(&traits.Count{}, "added", "removed"),
		),
	}
}

func InitialCount() *traits.Count {
	return &traits.Count{
		ResetTime: timestamppb.Now(),
	}
}

func (t *MemoryDevice) GetCount(_ context.Context, req *traits.GetCountRequest) (*traits.Count, error) {
	return t.count.Get(resource.WithReadMask(req.ReadMask)).(*traits.Count), nil
}

func (t *MemoryDevice) ResetCount(_ context.Context, request *traits.ResetCountRequest) (*traits.Count, error) {
	rt := request.ResetTime
	if rt == nil {
		rt = timestamppb.Now()
	}
	res, err := t.count.Set(&traits.Count{Added: 0, Removed: 0, ResetTime: rt}, resource.WithAllFieldsWritable())
	return res.(*traits.Count), err
}

func (t *MemoryDevice) UpdateCount(_ context.Context, request *traits.UpdateCountRequest) (*traits.Count, error) {
	res, err := t.count.Set(request.Count, resource.WithUpdateMask(request.UpdateMask), resource.InterceptBefore(func(old, value proto.Message) {
		if request.Delta {
			tOld := old.(*traits.Count)
			tValue := value.(*traits.Count)
			tValue.Added += tOld.Added
			tValue.Removed += tOld.Removed
		}
	}))
	return res.(*traits.Count), err
}

func (t *MemoryDevice) PullCounts(request *traits.PullCountsRequest, server traits.CountApi_PullCountsServer) error {
	for event := range t.count.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullCountsResponse_Change{
			Name:  request.Name,
			Count: event.Value.(*traits.Count),
		}
		err := server.Send(&traits.PullCountsResponse{
			Changes: []*traits.PullCountsResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
