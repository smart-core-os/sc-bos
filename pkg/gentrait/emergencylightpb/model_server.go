package emergencylightpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	emergencylightpb.UnimplementedEmergencyLightApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	emergencylightpb.RegisterEmergencyLightApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetTestResultSet(context.Context, *emergencylightpb.GetTestResultSetRequest) (*emergencylightpb.TestResultSet, error) {
	return m.model.GetTestResultSet(), nil
}

func (m *ModelServer) StartFunctionTest(context.Context, *emergencylightpb.StartEmergencyTestRequest) (*emergencylightpb.StartEmergencyTestResponse, error) {
	m.model.RunFunctionTest()
	return &emergencylightpb.StartEmergencyTestResponse{}, nil
}

func (m *ModelServer) StartDurationTest(context.Context, *emergencylightpb.StartEmergencyTestRequest) (*emergencylightpb.StartEmergencyTestResponse, error) {
	m.model.RunDurationTest()
	return &emergencylightpb.StartEmergencyTestResponse{}, nil
}

func (m *ModelServer) StopEmergencyTest(context.Context, *emergencylightpb.StopEmergencyTestsRequest) (*emergencylightpb.StopEmergencyTestsResponse, error) {
	// No-op for this model, as tests are run immediately
	return &emergencylightpb.StopEmergencyTestsResponse{}, nil
}

func (m *ModelServer) PullTestResultSets(request *emergencylightpb.PullTestResultRequest, server grpc.ServerStreamingServer[emergencylightpb.PullTestResultsResponse]) error {
	for change := range m.model.PullTestResults(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &emergencylightpb.PullTestResultsResponse{Changes: []*emergencylightpb.PullTestResultsResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(change.ChangeTime),
			TestResult: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
