package healthpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/util/page"
)

// ModelServer is a HealthApiServer backed by a Model.
type ModelServer struct {
	gen.UnimplementedHealthApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (m *ModelServer) ListHealthChecks(_ context.Context, request *gen.ListHealthChecksRequest) (*gen.ListHealthChecksResponse, error) {
	items, totalSize, nextPageToken, err := page.List(request, (*gen.HealthCheck).GetId, func() []*gen.HealthCheck {
		return m.model.ListHealthChecks(listOptions(request)...)
	})
	if err != nil {
		return nil, err
	}
	return &gen.ListHealthChecksResponse{
		HealthChecks:  items,
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}, nil
}

func (m *ModelServer) PullHealthChecks(request *gen.PullHealthChecksRequest, g grpc.ServerStreamingServer[gen.PullHealthChecksResponse]) error {
	for change := range m.model.PullHealthChecks(g.Context(), pullOptions(request)...) {
		err := g.Send(&gen.PullHealthChecksResponse{Changes: []*gen.PullHealthChecksResponse_Change{
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

func (m *ModelServer) GetHealthCheck(_ context.Context, request *gen.GetHealthCheckRequest) (*gen.HealthCheck, error) {
	return m.model.GetHealthCheck(request.GetId(), getOptions(request)...)
}

func (m *ModelServer) PullHealthCheck(request *gen.PullHealthCheckRequest, g grpc.ServerStreamingServer[gen.PullHealthCheckResponse]) error {
	for change := range m.model.PullHealthCheck(g.Context(), request.GetId(), pullOptions(request)...) {
		err := g.Send(&gen.PullHealthCheckResponse{Changes: []*gen.PullHealthCheckResponse_Change{{
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
