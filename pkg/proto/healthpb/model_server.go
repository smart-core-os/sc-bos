package healthpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/util/page"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer is a HealthApiServer backed by a Model.
type ModelServer struct {
	UnimplementedHealthApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (m *ModelServer) ListHealthChecks(_ context.Context, request *ListHealthChecksRequest) (*ListHealthChecksResponse, error) {
	items, totalSize, nextPageToken, err := page.List(request, (*HealthCheck).GetId, func() []*HealthCheck {
		return m.model.ListHealthChecks(listOptions(request)...)
	})
	if err != nil {
		return nil, err
	}
	return &ListHealthChecksResponse{
		HealthChecks:  items,
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}, nil
}

func (m *ModelServer) PullHealthChecks(request *PullHealthChecksRequest, g grpc.ServerStreamingServer[PullHealthChecksResponse]) error {
	for change := range m.model.PullHealthChecks(g.Context(), pullOptions(request)...) {
		err := g.Send(&PullHealthChecksResponse{Changes: []*PullHealthChecksResponse_Change{
			{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				Type:       change.ChangeType,
				OldValue:   change.OldValue,
				NewValue:   change.NewValue,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) GetHealthCheck(_ context.Context, request *GetHealthCheckRequest) (*HealthCheck, error) {
	return m.model.GetHealthCheck(request.GetId(), getOptions(request)...)
}

func (m *ModelServer) PullHealthCheck(request *PullHealthCheckRequest, g grpc.ServerStreamingServer[PullHealthCheckResponse]) error {
	for change := range m.model.PullHealthCheck(g.Context(), request.GetId(), pullOptions(request)...) {
		err := g.Send(&PullHealthCheckResponse{Changes: []*PullHealthCheckResponse_Change{{
			Name:        request.Name,
			ChangeTime:  timestamppb.New(change.ChangeTime),
			HealthCheck: change.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return nil
}

type readRequest interface {
	GetReadMask() *fieldmaskpb.FieldMask
}

type pullRequest interface {
	readRequest
	GetUpdatesOnly() bool
}

func getOptions(req readRequest, opts ...resource.ReadOption) []resource.ReadOption {
	return append(opts, resource.WithReadMask(req.GetReadMask()), resource.WithUpdatesOnly(false))
}

func listOptions(req readRequest, opts ...resource.ReadOption) []resource.ReadOption {
	return append(opts, resource.WithReadMask(req.GetReadMask()))
}

func pullOptions(req pullRequest, opts ...resource.ReadOption) []resource.ReadOption {
	return append(opts, resource.WithReadMask(req.GetReadMask()), resource.WithUpdatesOnly(req.GetUpdatesOnly()))
}
