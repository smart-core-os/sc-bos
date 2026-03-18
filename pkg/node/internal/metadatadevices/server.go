package metadatadevices

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// Server implements the MetadataApi backed by a devicespb.Collection.
type Server struct {
	metadatapb.UnimplementedMetadataApiServer
	devices Collection
}

// Collection contains devices keyed by their name.
type Collection interface {
	GetDevice(name string, opts ...resource.ReadOption) (*devicespb.Device, error)
	PullDevice(ctx context.Context, name string, opts ...resource.ReadOption) <-chan devicespb.DeviceChange
}

func NewServer(devices Collection) *Server {
	return &Server{
		devices: devices,
	}
}

func (s Server) GetMetadata(_ context.Context, request *metadatapb.GetMetadataRequest) (*metadatapb.Metadata, error) {
	device, err := s.devices.GetDevice(request.Name)
	if err != nil {
		return nil, err
	}
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	return filter.FilterClone(device.Metadata).(*metadatapb.Metadata), nil
}

func (s Server) PullMetadata(request *metadatapb.PullMetadataRequest, g grpc.ServerStreamingServer[metadatapb.PullMetadataResponse]) error {
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	for change := range s.devices.PullDevice(g.Context(), request.Name, resource.WithUpdatesOnly(request.UpdatesOnly)) {
		mdChange := deviceChangeToProto(request.Name, change, filter)
		err := g.Send(&metadatapb.PullMetadataResponse{Changes: []*metadatapb.PullMetadataResponse_Change{mdChange}})
		if err != nil {
			return err
		}
	}
	return nil
}

func deviceChangeToProto(name string, c devicespb.DeviceChange, filter *masks.ResponseFilter) *metadatapb.PullMetadataResponse_Change {
	res := &metadatapb.PullMetadataResponse_Change{
		Name:       name,
		ChangeTime: timestamppb.New(c.ChangeTime),
	}
	if c.Value.GetMetadata() == nil {
		res.Metadata = &metadatapb.Metadata{}
	} else {
		res.Metadata = filter.FilterClone(c.Value.GetMetadata()).(*metadatapb.Metadata)
	}
	return res
}
